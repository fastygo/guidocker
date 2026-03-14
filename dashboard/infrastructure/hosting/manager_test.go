package hosting

import (
	"context"
	"dashboard/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNginxHostManager_ApplyRemoveAndLifecycle(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()
	manager := NewNginxHostManagerWithOptions("nginx", tempDir)
	manager.runScript = func(_ context.Context, name string, args ...string) ([]byte, error) {
		return []byte(name + " " + strings.Join(args, " ")), nil
	}

	app := &domain.App{
		ID:             "app-1",
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          false,
	}
	if err := manager.ApplyRouting(ctx, app, domain.PlatformSettings{}); err != nil {
		t.Fatalf("ApplyRouting() error = %v", err)
	}

	configPath := filepath.Join(tempDir, "paas-app-app-1.conf")
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if got := string(contents); !strings.Contains(got, "server_name app.example.com") {
		t.Fatalf("expected config to contain server name, got %q", got)
	}
	if got := string(contents); !strings.Contains(got, "127.0.0.1:8080") {
		t.Fatalf("expected config to contain upstream target, got %q", got)
	}

	if err := manager.ValidateRouting(ctx); err != nil {
		t.Fatalf("ValidateRouting() error = %v", err)
	}
	if err := manager.ReloadRouting(ctx); err != nil {
		t.Fatalf("ReloadRouting() error = %v", err)
	}

	if err := manager.RemoveRouting(ctx, app, domain.PlatformSettings{}); err != nil {
		t.Fatalf("RemoveRouting() error = %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected config file to be removed, stat error = %v", err)
	}
}

func TestCertbotManager_EnsureCertificate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	called := 0
	var command []string
	manager := NewCertbotManagerWithBinary("certbot")
	manager.runScript = func(_ context.Context, name string, args ...string) ([]byte, error) {
		called++
		command = append([]string{name}, args...)
		return []byte("issued"), nil
	}

	err := manager.EnsureCertificate(ctx, domain.PlatformSettings{
		CertbotEnabled:       true,
		CertbotEmail:         "ops@example.com",
		CertbotTermsAccepted: true,
		CertbotStaging:       true,
	}, "app.example.com")
	if err != nil {
		t.Fatalf("EnsureCertificate() error = %v", err)
	}
	if called != 1 {
		t.Fatalf("expected runScript called once, got %d", called)
	}
	if len(command) < 3 || command[0] != "certbot" {
		t.Fatalf("unexpected command: %v", command)
	}
	if !strings.Contains(strings.Join(command, " "), "--test-cert") {
		t.Fatalf("expected test-cert argument, got %v", command)
	}
}

func TestCertbotManager_RemoveCertificate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	manager := NewCertbotManagerWithBinary("certbot")
	manager.runScript = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "certbot" {
			t.Fatalf("expected certbot binary, got %q", name)
		}
		if len(args) != 4 || args[0] != "delete" || args[1] != "--non-interactive" || args[2] != "--cert-name" || args[3] != "app.example.com" {
			t.Fatalf("unexpected args: %v", args)
		}
		return []byte("deleted"), nil
	}

	if err := manager.RemoveCertificate(ctx, "app.example.com"); err != nil {
		t.Fatalf("RemoveCertificate() error = %v", err)
	}
}

