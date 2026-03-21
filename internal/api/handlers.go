package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/damn8daniel/observability-platform/internal/ingestion"
	"github.com/damn8daniel/observability-platform/internal/storage"
	"github.com/damn8daniel/observability-platform/internal/tenant"
)

// Handlers holds all HTTP API handler dependencies.
type Handlers struct {
	store     *storage.ClickHouseStore
	logBuf    *ingestion.LogBuffer
	spanBuf   *ingestion.SpanBuffer
	metricBuf *ingestion.MetricBuffer
	logger    *slog.Logger
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(
	store *storage.ClickHouseStore,
	logBuf *ingestion.LogBuffer,
	spanBuf *ingestion.SpanBuffer,
	metricBuf *ingestion.MetricBuffer,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{
		store:     store,
		logBuf:    logBuf,
		spanBuf:   spanBuf,
		metricBuf: metricBuf,
		logger:    logger,
	}
}

// --- Log Endpoints ---

// IngestLogs handles POST /api/v1/logs
func (h *Handlers) IngestLogs(w http.ResponseWriter, r *http.Request) {
	var entries []storage.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	tenantID := tenant.FromContext(r.Context())
	for i := range entries {
		if entries[i].ID == "" {
			entries[i].ID = uuid.New().String()
		}
		entries[i].TenantID = tenantID
		if entries[i].Timestamp.IsZero() {
			entries[i].Timestamp = time.Now()
		}
	}

	h.logBuf.PushBatch(entries)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"accepted": len(entries),
	})
}

// QueryLogs handles GET /api/v1/logs
func (h *Handlers) QueryLogs(w http.ResponseWriter, r *http.Request) {
	q := storage.LogQuery{
		TenantID: tenant.FromContext(r.Context()),
		Query:    r.URL.Query().Get("q"),
		Level:    r.URL.Query().Get("level"),
		Service:  r.URL.Query().Get("service"),
		TraceID:  r.URL.Query().Get("trace_id"),
		OrderBy:  r.URL.Query().Get("order_by"),
		OrderDir: r.URL.Query().Get("order_dir"),
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		q.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		q.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("start"); v != "" {
		q.StartTime, _ = time.Parse(time.RFC3339, v)
	}
	if v := r.URL.Query().Get("end"); v != "" {
		q.EndTime, _ = time.Parse(time.RFC3339, v)
	}

	logs, total, err := h.store.QueryLogs(r.Context(), q)
	if err != nil {
		h.logger.Error("query logs failed", "error", err)
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"total": total,
	})
}

// AggregateLogs handles GET /api/v1/logs/aggregate
func (h *Handlers) AggregateLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.FromContext(r.Context())
	field := r.URL.Query().Get("field")
	if field == "" {
		writeError(w, http.StatusBadRequest, "field parameter required")
		return
	}

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	if v := r.URL.Query().Get("start"); v != "" {
		start, _ = time.Parse(time.RFC3339, v)
	}
	if v := r.URL.Query().Get("end"); v != "" {
		end, _ = time.Parse(time.RFC3339, v)
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	result, err := h.store.AggregateLogsByField(r.Context(), tenantID, field, start, end, limit)
	if err != nil {
		h.logger.Error("aggregate logs failed", "error", err)
		writeError(w, http.StatusInternalServerError, "aggregation failed")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Trace Endpoints ---

// IngestSpans handles POST /api/v1/traces
func (h *Handlers) IngestSpans(w http.ResponseWriter, r *http.Request) {
	var spans []storage.Span
	if err := json.NewDecoder(r.Body).Decode(&spans); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	tenantID := tenant.FromContext(r.Context())
	for i := range spans {
		spans[i].TenantID = tenantID
	}

	h.spanBuf.PushBatch(spans)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"accepted": len(spans),
	})
}

// QueryTraces handles GET /api/v1/traces
func (h *Handlers) QueryTraces(w http.ResponseWriter, r *http.Request) {
	q := storage.TraceQuery{
		TenantID:  tenant.FromContext(r.Context()),
		TraceID:   r.URL.Query().Get("trace_id"),
		Service:   r.URL.Query().Get("service"),
		Operation: r.URL.Query().Get("operation"),
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		q.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("start"); v != "" {
		q.StartTime, _ = time.Parse(time.RFC3339, v)
	}
	if v := r.URL.Query().Get("end"); v != "" {
		q.EndTime, _ = time.Parse(time.RFC3339, v)
	}
	if v := r.URL.Query().Get("min_duration"); v != "" {
		d, _ := time.ParseDuration(v)
		q.MinDuration = d
	}
	if v := r.URL.Query().Get("max_duration"); v != "" {
		d, _ := time.ParseDuration(v)
		q.MaxDuration = d
	}

	spans, err := h.store.QueryTraces(r.Context(), q)
	if err != nil {
		h.logger.Error("query traces failed", "error", err)
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"spans": spans,
	})
}

// GetTrace handles GET /api/v1/traces/{traceID}
func (h *Handlers) GetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceID")
	tenantID := tenant.FromContext(r.Context())

	spans, err := h.store.QueryTraces(r.Context(), storage.TraceQuery{
		TenantID: tenantID,
		TraceID:  traceID,
		Limit:    1000,
	})
	if err != nil {
		h.logger.Error("get trace failed", "error", err)
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"trace_id": traceID,
		"spans":    spans,
	})
}

// --- Metrics Endpoints ---

// PushMetrics handles POST /api/v1/metrics
func (h *Handlers) PushMetrics(w http.ResponseWriter, r *http.Request) {
	var samples []storage.MetricSample
	if err := json.NewDecoder(r.Body).Decode(&samples); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	tenantID := tenant.FromContext(r.Context())
	for i := range samples {
		samples[i].TenantID = tenantID
		if samples[i].Timestamp.IsZero() {
			samples[i].Timestamp = time.Now()
		}
	}

	h.metricBuf.PushBatch(samples)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"accepted": len(samples),
	})
}

// --- Correlation ---

// GetCorrelation handles GET /api/v1/correlate/{traceID}
func (h *Handlers) GetCorrelation(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceID")
	tenantID := tenant.FromContext(r.Context())

	logs, spans, err := h.store.GetCorrelatedData(r.Context(), tenantID, traceID)
	if err != nil {
		h.logger.Error("correlation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "correlation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"trace_id": traceID,
		"logs":     logs,
		"spans":    spans,
	})
}

// --- Health ---

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
