package config

import (
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("SERVER_HOST", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("PAAS_PORT", "")
	t.Setenv("DASHBOARD_DATA_FILE", "")
	t.Setenv("PAAS_ADMIN_USER", "")
	t.Setenv("PAAS_ADMIN_PASS", "")
	t.Setenv("DASHBOARD_AUTH_DISABLED", "")
	t.Setenv("PAAS_AUTH_DISABLED", "")
	t.Setenv("STACKS_DIR", "")
	t.Setenv("BOLT_DB_FILE", "")

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
	if cfg.Auth.AdminUser != "admin" {
		t.Fatalf("expected default admin user admin, got %q", cfg.Auth.AdminUser)
	}
	if cfg.Auth.AdminPass != "admin@123" {
		t.Fatalf("expected default admin pass admin@123, got %q", cfg.Auth.AdminPass)
	}
	if cfg.Auth.Disabled {
		t.Fatalf("expected auth enabled by default")
	}
	if filepath.ToSlash(cfg.Stacks.Dir) != "/opt/stacks" {
		t.Fatalf("expected default stacks dir /opt/stacks, got %q", cfg.Stacks.Dir)
	}
	if filepath.ToSlash(cfg.Stacks.DBFile) != "/opt/stacks/.paas.db" {
		t.Fatalf("expected default BoltDB file /opt/stacks/.paas.db, got %q", cfg.Stacks.DBFile)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("SERVER_HOST", "0.0.0.0")
	t.Setenv("PAAS_PORT", "9090")
	t.Setenv("DASHBOARD_DATA_FILE", "/tmp/dashboard.json")
	t.Setenv("PAAS_ADMIN_USER", "root")
	t.Setenv("PAAS_ADMIN_PASS", "secret")
	t.Setenv("DASHBOARD_AUTH_DISABLED", "true")
	t.Setenv("STACKS_DIR", "/srv/stacks")
	t.Setenv("BOLT_DB_FILE", "/srv/stacks/apps.db")

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
	if cfg.Auth.AdminUser != "root" || cfg.Auth.AdminPass != "secret" {
		t.Fatalf("expected auth overrides to be applied, got %+v", cfg.Auth)
	}
	if !cfg.Auth.Disabled {
		t.Fatalf("expected auth disabled from env flag, got enabled")
	}
	if cfg.Stacks.Dir != "/srv/stacks" || cfg.Stacks.DBFile != "/srv/stacks/apps.db" {
		t.Fatalf("expected stack overrides to be applied, got %+v", cfg.Stacks)
	}
}

func TestLoad_InvalidPortFallsBackToDefault(t *testing.T) {
	t.Setenv("PAAS_PORT", "not-a-number")
	cfg := Load()
	if cfg.Server.Port != 3000 {
		t.Fatalf("expected fallback port 3000, got %d", cfg.Server.Port)
	}
}
