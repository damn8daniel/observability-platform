package tenant

import (
	"context"
	"fmt"
	"sync"
)

type contextKey string

const tenantKey contextKey = "tenant_id"

// Tenant represents an isolated tenant in the system.
type Tenant struct {
	ID          string
	Name        string
	RateLimitRPS int
	MaxLogSize  int64
	Enabled     bool
}

// Registry manages tenant lifecycle and lookup.
type Registry struct {
	mu      sync.RWMutex
	tenants map[string]*Tenant
}

// NewRegistry creates a new tenant registry.
func NewRegistry() *Registry {
	return &Registry{
		tenants: make(map[string]*Tenant),
	}
}

// Register adds a new tenant to the registry.
func (r *Registry) Register(t *Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tenants[t.ID]; exists {
		return fmt.Errorf("tenant %q already registered", t.ID)
	}

	r.tenants[t.ID] = t
	return nil
}

// Get retrieves a tenant by ID.
func (r *Registry) Get(id string) (*Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tenants[id]
	if !ok {
		return nil, fmt.Errorf("tenant %q not found", id)
	}

	if !t.Enabled {
		return nil, fmt.Errorf("tenant %q is disabled", id)
	}

	return t, nil
}

// List returns all registered tenants.
func (r *Registry) List() []*Tenant {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Tenant, 0, len(r.tenants))
	for _, t := range r.tenants {
		result = append(result, t)
	}
	return result
}

// Delete removes a tenant from the registry.
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tenants[id]; !exists {
		return fmt.Errorf("tenant %q not found", id)
	}

	delete(r.tenants, id)
	return nil
}

// WithTenantID adds a tenant ID to the context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// FromContext extracts the tenant ID from context.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(tenantKey).(string)
	return id
}
