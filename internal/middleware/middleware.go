package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spdedsec/gopherd/internal/model"
)

type contextKey string

const requestIDKey contextKey = "request_id"
const userKey contextKey = "user"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = randomID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}
func GetRequestID(ctx context.Context) string { v, _ := ctx.Value(requestIDKey).(string); return v }
func WithUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}
func GetUser(ctx context.Context) (model.User, bool) {
	v, ok := ctx.Value(userKey).(model.User)
	return v, ok
}
func Recover(logger *slog.Logger, metrics *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				logger.Error("panic recovered", "panic", v, "request_id", GetRequestID(r.Context()))
				metrics.errors.Add(1)
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error", "request_id": GetRequestID(r.Context())})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}
func Logging(logger *slog.Logger, metrics *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(rw, r)
		if rw.status == 0 {
			rw.status = 200
		}
		logger.Info("http_request", "request_id", GetRequestID(r.Context()), "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration_ms", time.Since(start).Milliseconds(), "bytes", rw.bytes, "remote", r.RemoteAddr)
		metrics.observe(rw.status, time.Since(start))
	})
}

type Metrics struct {
	requests     atomic.Uint64
	errors       atomic.Uint64
	totalLatency atomic.Uint64
}

func NewMetrics() *Metrics { return &Metrics{} }
func (m *Metrics) observe(status int, d time.Duration) {
	m.requests.Add(1)
	m.totalLatency.Add(uint64(d.Microseconds()))
	if status >= 500 {
		m.errors.Add(1)
	}
}
func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	req := m.requests.Load()
	errc := m.errors.Load()
	lat := m.totalLatency.Load()
	avg := uint64(0)
	if req > 0 {
		avg = lat / req
	}
	_, _ = w.Write([]byte("# HELP gopherd_http_requests_total Total HTTP requests.\n# TYPE gopherd_http_requests_total counter\ngopherd_http_requests_total " + itoa(req) + "\n# HELP gopherd_http_errors_total Total HTTP 5xx responses.\n# TYPE gopherd_http_errors_total counter\ngopherd_http_errors_total " + itoa(errc) + "\n# HELP gopherd_http_average_latency_microseconds Average request latency.\n# TYPE gopherd_http_average_latency_microseconds gauge\ngopherd_http_average_latency_microseconds " + itoa(avg) + "\n"))
}
func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := 20
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

// RateLimiter is a small in-process token bucket. It is intentionally single-instance; use an
// external gateway/limiter when horizontally scaling beyond one application instance.
type RateLimiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucket
}
type bucket struct {
	tokens float64
	last   time.Time
}

func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{rate: rate, burst: float64(burst), buckets: make(map[string]*bucket)}
}
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)
		if !l.allow(key) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "rate limit exceeded", "request_id": GetRequestID(r.Context())})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (l *RateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[key]
	if b == nil {
		l.buckets[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func CORS(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowedOrigin(origin, origins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func allowedOrigin(origin string, origins []string) bool {
	for _, o := range origins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func randomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
func Bearer(r *http.Request) string {
	v := r.Header.Get("Authorization")
	parts := strings.Fields(v)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
