package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/damn8daniel/observability-platform/internal/config"
	"github.com/damn8daniel/observability-platform/internal/tenant"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// HTTPLogging returns middleware that logs each HTTP request.
func HTTPLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			logger.Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration", time.Since(start),
				"remote", r.RemoteAddr,
			)
		})
	}
}

// HTTPRecovery returns middleware that recovers from panics.
func HTTPRecovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("HTTP panic recovered",
						"panic", rec,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// HTTPTenant extracts the tenant ID from the configured header and injects it into the context.
func HTTPTenant(cfg config.TenancyConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := cfg.DefaultTenant

			if cfg.Enabled {
				headerVal := r.Header.Get(cfg.HeaderName)
				if headerVal != "" {
					tenantID = headerVal
				} else if cfg.DefaultTenant == "" {
					http.Error(w, "missing tenant ID header", http.StatusUnauthorized)
					return
				}
			}

			ctx := tenant.WithTenantID(r.Context(), tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS returns middleware that sets permissive CORS headers for the dashboard.
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
