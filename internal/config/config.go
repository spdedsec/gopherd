package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment     string
	ListenAddr      string
	DatabaseURL     string
	DBMaxConns      int32
	DBMinConns      int32
	SessionTTL      time.Duration
	MaxBodyBytes    int64
	RateLimitRPS    float64
	RateLimitBurst  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	CORSOrigins     []string
	LogLevel        slog.Level
}

func Load() (Config, error) {
	c := Config{
		Environment:     env("ENVIRONMENT", "development"),
		ListenAddr:      env("LISTEN_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DBMaxConns:      int32(envInt("DB_MAX_CONNS", 10)),
		DBMinConns:      int32(envInt("DB_MIN_CONNS", 2)),
		SessionTTL:      envDuration("SESSION_TTL", 168*time.Hour),
		MaxBodyBytes:    int64(envInt("MAX_BODY_BYTES", 1<<20)),
		RateLimitRPS:    envFloat("RATE_LIMIT_RPS", 10),
		RateLimitBurst:  envInt("RATE_LIMIT_BURST", 20),
		ReadTimeout:     envDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    envDuration("WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     envDuration("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		CORSOrigins:     splitCSV(env("CORS_ORIGINS", "")),
		LogLevel:        slog.LevelInfo,
	}
	if level := strings.ToLower(env("LOG_LEVEL", "info")); level == "debug" {
		c.LogLevel = slog.LevelDebug
	} else if level == "warn" {
		c.LogLevel = slog.LevelWarn
	} else if level == "error" {
		c.LogLevel = slog.LevelError
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if c.DBMinConns < 0 || c.DBMaxConns < c.DBMinConns || c.DBMaxConns < 1 {
		return Config{}, fmt.Errorf("invalid database pool settings")
	}
	if c.RateLimitRPS <= 0 || c.RateLimitBurst < 1 {
		return Config{}, fmt.Errorf("invalid rate limit settings")
	}
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return fallback
}
func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
func splitCSV(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
