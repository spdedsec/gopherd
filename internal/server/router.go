package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spdedsec/gopherd/internal/config"
	"github.com/spdedsec/gopherd/internal/handler"
	"github.com/spdedsec/gopherd/internal/middleware"
	"github.com/spdedsec/gopherd/internal/service"
)

type Dependencies struct {
	Config      config.Config
	Auth        *handler.AuthHandler
	Tasks       *handler.TaskHandler
	Health      *handler.HealthHandler
	Metrics     *middleware.Metrics
	RateLimiter *middleware.RateLimiter
	AuthService *service.AuthService
}

func NewRouter(d Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.CORS(d.Config.CORSOrigins))
	r.Use(d.RateLimiter.Middleware)
	r.Get("/health/live", d.Health.Live)
	r.Get("/health/ready", d.Health.Ready)
	r.Handle("/metrics", d.Metrics)
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("gopherd\n"))
	})
	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/register", d.Auth.Register)
		api.Post("/auth/login", d.Auth.Login)
		api.Group(func(protected chi.Router) {
			protected.Use(authMiddleware(d.AuthService))
			protected.Post("/auth/logout", d.Auth.Logout)
			protected.Get("/me", d.Auth.Me)
			protected.Get("/tasks", d.Tasks.List)
			protected.Post("/tasks", d.Tasks.Create)
			protected.Get("/tasks/{id}", d.Tasks.Get)
			protected.Put("/tasks/{id}", d.Tasks.Update)
			protected.Delete("/tasks/{id}", d.Tasks.Delete)
		})
	})
	return r
}
func authMiddleware(s *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := middleware.Bearer(r)
			u, err := s.Authenticate(r.Context(), token)
			if err != nil {
				writeUnauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(middleware.WithUser(r.Context(), u)))
		})
	}
}
func writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized","request_id":"` + middleware.GetRequestID(r.Context()) + `"}\n`))
}
