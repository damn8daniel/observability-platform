package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/damn8daniel/observability-platform/internal/alerting"
	"github.com/damn8daniel/observability-platform/internal/storage"
	"github.com/damn8daniel/observability-platform/internal/tenant"
)

// AlertHandlers manages alert rule CRUD and alert queries.
type AlertHandlers struct {
	engine *alerting.Engine
}

// NewAlertHandlers creates alert API handlers.
func NewAlertHandlers(engine *alerting.Engine) *AlertHandlers {
	return &AlertHandlers{engine: engine}
}

// CreateAlertRule handles POST /api/v1/alerts/rules
func (ah *AlertHandlers) CreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule storage.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	rule.ID = uuid.New().String()
	rule.TenantID = tenant.FromContext(r.Context())
	rule.CreatedAt = time.Now()
	rule.Enabled = true

	ah.engine.AddRule(rule)

	writeJSON(w, http.StatusCreated, rule)
}

// ListAlertRules handles GET /api/v1/alerts/rules
func (ah *AlertHandlers) ListAlertRules(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.FromContext(r.Context())
	rules := ah.engine.GetRules(tenantID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
	})
}

// DeleteAlertRule handles DELETE /api/v1/alerts/rules/{ruleID}
func (ah *AlertHandlers) DeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleID")
	ah.engine.RemoveRule(ruleID)
	w.WriteHeader(http.StatusNoContent)
}

// ListAlerts handles GET /api/v1/alerts
func (ah *AlertHandlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.FromContext(r.Context())
	alerts := ah.engine.GetAlerts(tenantID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
	})
}
