package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/damn8daniel/observability-platform/internal/alerting"
	"github.com/damn8daniel/observability-platform/internal/config"
	"github.com/damn8daniel/observability-platform/internal/ingestion"
	"github.com/damn8daniel/observability-platform/internal/middleware"
	"github.com/damn8daniel/observability-platform/internal/storage"
)

// NewRouter creates the HTTP router with all routes registered.
func NewRouter(
	cfg config.Config,
	store *storage.ClickHouseStore,
	logBuf *ingestion.LogBuffer,
	spanBuf *ingestion.SpanBuffer,
	metricBuf *ingestion.MetricBuffer,
	alertEngine *alerting.Engine,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.HTTPLogging(logger))
	r.Use(middleware.HTTPRecovery(logger))
	r.Use(middleware.CORS())
	r.Use(chimw.Compress(5))

	h := NewHandlers(store, logBuf, spanBuf, metricBuf, logger)
	dashH := NewDashboardHandlers()
	alertH := NewAlertHandlers(alertEngine)

	// Health & metrics
	r.Get("/health", h.HealthCheck)
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Tenant middleware for all API routes
		r.Use(middleware.HTTPTenant(cfg.Tenancy))

		// Logs
		r.Post("/logs", h.IngestLogs)
		r.Get("/logs", h.QueryLogs)
		r.Get("/logs/aggregate", h.AggregateLogs)

		// Traces
		r.Post("/traces", h.IngestSpans)
		r.Get("/traces", h.QueryTraces)
		r.Get("/traces/{traceID}", h.GetTrace)

		// Metrics
		r.Post("/metrics", h.PushMetrics)

		// Correlation
		r.Get("/correlate/{traceID}", h.GetCorrelation)

		// Dashboards
		r.Post("/dashboards", dashH.CreateDashboard)
		r.Get("/dashboards", dashH.ListDashboards)
		r.Get("/dashboards/{dashboardID}", dashH.GetDashboard)
		r.Put("/dashboards/{dashboardID}", dashH.UpdateDashboard)
		r.Delete("/dashboards/{dashboardID}", dashH.DeleteDashboard)

		// Alerts
		r.Post("/alerts/rules", alertH.CreateAlertRule)
		r.Get("/alerts/rules", alertH.ListAlertRules)
		r.Delete("/alerts/rules/{ruleID}", alertH.DeleteAlertRule)
		r.Get("/alerts", alertH.ListAlerts)
	})

	return r
}
