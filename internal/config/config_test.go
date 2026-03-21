package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.Server.HTTPAddr, ":8080")
	}

	if cfg.Server.GRPCAddr != ":9090" {
		t.Errorf("GRPCAddr = %q, want %q", cfg.Server.GRPCAddr, ":9090")
	}

	if cfg.Retention.LogsTTL != 30*24*time.Hour {
		t.Errorf("LogsTTL = %v, want 30 days", cfg.Retention.LogsTTL)
	}

	if cfg.Tenancy.DefaultTenant != "default" {
		t.Errorf("DefaultTenant = %q, want %q", cfg.Tenancy.DefaultTenant, "default")
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
