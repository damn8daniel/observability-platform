package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/damn8daniel/observability-platform/internal/storage"
	"github.com/damn8daniel/observability-platform/internal/tenant"
)

// DashboardHandlers manages dashboard CRUD operations.
// Note: In a full implementation, dashboards would be stored in ClickHouse.
// Here we provide the API contract and in-memory storage for demonstration.
type DashboardHandlers struct {
	dashboards map[string]*storage.Dashboard // tenant:id -> dashboard
}

// NewDashboardHandlers creates dashboard handlers.
func NewDashboardHandlers() *DashboardHandlers {
	return &DashboardHandlers{
		dashboards: make(map[string]*storage.Dashboard),
	}
}

// CreateDashboard handles POST /api/v1/dashboards
func (dh *DashboardHandlers) CreateDashboard(w http.ResponseWriter, r *http.Request) {
	var dash storage.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dash); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	dash.ID = uuid.New().String()
	dash.TenantID = tenant.FromContext(r.Context())
	dash.CreatedAt = time.Now()
	dash.UpdatedAt = time.Now()

	key := dash.TenantID + ":" + dash.ID
	dh.dashboards[key] = &dash

	writeJSON(w, http.StatusCreated, dash)
}

// ListDashboards handles GET /api/v1/dashboards
func (dh *DashboardHandlers) ListDashboards(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.FromContext(r.Context())
	var result []storage.Dashboard

	for _, d := range dh.dashboards {
		if d.TenantID == tenantID {
			result = append(result, *d)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"dashboards": result,
	})
}

// GetDashboard handles GET /api/v1/dashboards/{dashboardID}
func (dh *DashboardHandlers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	dashID := chi.URLParam(r, "dashboardID")
	tenantID := tenant.FromContext(r.Context())
	key := tenantID + ":" + dashID

	dash, ok := dh.dashboards[key]
	if !ok {
		writeError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	writeJSON(w, http.StatusOK, dash)
}

// UpdateDashboard handles PUT /api/v1/dashboards/{dashboardID}
func (dh *DashboardHandlers) UpdateDashboard(w http.ResponseWriter, r *http.Request) {
	dashID := chi.URLParam(r, "dashboardID")
	tenantID := tenant.FromContext(r.Context())
	key := tenantID + ":" + dashID

	existing, ok := dh.dashboards[key]
	if !ok {
		writeError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	var update storage.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	existing.Name = update.Name
	existing.Panels = update.Panels
	existing.UpdatedAt = time.Now()

	writeJSON(w, http.StatusOK, existing)
}

// DeleteDashboard handles DELETE /api/v1/dashboards/{dashboardID}
func (dh *DashboardHandlers) DeleteDashboard(w http.ResponseWriter, r *http.Request) {
	dashID := chi.URLParam(r, "dashboardID")
	tenantID := tenant.FromContext(r.Context())
	key := tenantID + ":" + dashID

	if _, ok := dh.dashboards[key]; !ok {
		writeError(w, http.StatusNotFound, "dashboard not found")
		return
	}

	delete(dh.dashboards, key)
	w.WriteHeader(http.StatusNoContent)
}
