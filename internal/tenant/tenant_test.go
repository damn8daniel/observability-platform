package tenant

import (
	"context"
	"testing"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()

	tenant := &Tenant{
		ID:      "test-tenant",
		Name:    "Test",
		Enabled: true,
	}

	if err := r.Register(tenant); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate registration should fail
	if err := r.Register(tenant); err == nil {
		t.Fatal("expected error for duplicate registration")
	}

	got, err := r.Get("test-tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != tenant.ID {
		t.Errorf("got ID %q, want %q", got.ID, tenant.ID)
	}

	// Non-existent tenant
	_, err = r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tenant")
	}
}

func TestRegistry_DisabledTenant(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Tenant{ID: "disabled", Enabled: false})

	_, err := r.Get("disabled")
	if err == nil {
		t.Fatal("expected error for disabled tenant")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Tenant{ID: "a", Enabled: true})
	_ = r.Register(&Tenant{ID: "b", Enabled: true})

	list := r.List()
	if len(list) != 2 {
		t.Errorf("got %d tenants, want 2", len(list))
	}
}

func TestRegistry_Delete(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&Tenant{ID: "del", Enabled: true})

	if err := r.Delete("del"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := r.Delete("nonexistent"); err == nil {
		t.Fatal("expected error for deleting nonexistent tenant")
	}
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenantID(ctx, "ctx-tenant")

	got := FromContext(ctx)
	if got != "ctx-tenant" {
		t.Errorf("got %q, want %q", got, "ctx-tenant")
	}

	// Empty context
	empty := FromContext(context.Background())
	if empty != "" {
		t.Errorf("got %q, want empty string", empty)
	}
}
