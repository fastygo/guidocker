package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("SERVER_HOST", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("DASHBOARD_DATA_FILE", "")

	cfg := Load()
	if cfg.Server.Host != "localhost" {
		t.Fatalf("expected default host localhost, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 3000 {
		t.Fatalf("expected default port 3000, got %d", cfg.Server.Port)
	}
	if cfg.Data.DashboardFile != "data/dashboard.json" {
		t.Fatalf("expected default dashboard data file, got %q", cfg.Data.DashboardFile)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("SERVER_HOST", "0.0.0.0")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DASHBOARD_DATA_FILE", "/tmp/dashboard.json")

	cfg := Load()
	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Data.DashboardFile != "/tmp/dashboard.json" {
		t.Fatalf("expected dashboard file override, got %q", cfg.Data.DashboardFile)
	}
}

func TestLoad_InvalidPortFallsBackToDefault(t *testing.T) {
	t.Setenv("SERVER_PORT", "not-a-number")
	cfg := Load()
	if cfg.Server.Port != 3000 {
		t.Fatalf("expected fallback port 3000, got %d", cfg.Server.Port)
	}
}
