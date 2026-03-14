package hosting

import (
	"context"
	"errors"
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

func TestNginxHostManager_GeneratesHTTPRedirectWhenTLSReady(t *testing.T) {
	tempDir := t.TempDir()
	certDir := t.TempDir()
	ctx := context.Background()
	manager := NewNginxHostManagerWithOptions("nginx", tempDir)
	manager.runScript = func(_ context.Context, name string, args ...string) ([]byte, error) {
		return []byte(name + " " + strings.Join(args, " ")), nil
	}
	originalCertBasePath := certBasePath
	certBasePath = certDir
	defer func() {
		certBasePath = originalCertBasePath
	}()

	domainName := "app.example.com"
	certDomainPath := filepath.Join(certDir, domainName)
	if err := os.MkdirAll(certDomainPath, 0o755); err != nil {
		t.Fatalf("failed to create cert dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDomainPath, "fullchain.pem"), []byte("certificate"), 0o644); err != nil {
		t.Fatalf("failed to write fullchain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDomainPath, "privkey.pem"), []byte("private key"), 0o644); err != nil {
		t.Fatalf("failed to write privkey: %v", err)
	}

	app := &domain.App{
		ID:             "app-2",
		PublicDomain:    domainName,
		ProxyTargetPort: 8080,
		UseTLS:          true,
	}
	if err := manager.ApplyRouting(ctx, app, domain.PlatformSettings{}); err != nil {
		t.Fatalf("ApplyRouting() error = %v", err)
	}

	configPath := filepath.Join(tempDir, "paas-app-app-2.conf")
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	configText := string(contents)
	if !strings.Contains(configText, "return 301 https://$host$request_uri;") {
		t.Fatalf("expected HTTP redirect, got %q", configText)
	}
	if !strings.Contains(configText, "listen 443 ssl;") {
		t.Fatalf("expected HTTPS server block, got %q", configText)
	}
	if !strings.Contains(configText, "ssl_certificate "+filepath.Join(certDomainPath, "fullchain.pem")) {
		t.Fatalf("expected cert path, got %q", configText)
	}
}

func TestNginxHostManager_ApplyRouting_RequiresDomainAndPort(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()
	manager := NewNginxHostManagerWithOptions("nginx", tempDir)
	manager.runScript = func(_ context.Context, name string, args ...string) ([]byte, error) {
		t.Fatalf("unexpected runScript call: %s %v", name, args)
		return nil, nil
	}

	if err := manager.ApplyRouting(ctx, &domain.App{ID: "app-1", PublicDomain: "", ProxyTargetPort: 8080}, domain.PlatformSettings{}); err != nil {
		t.Fatalf("ApplyRouting() error = %v", err)
	}
	appConfigPath := filepath.Join(tempDir, "paas-app-app-1.conf")
	if _, err := os.Stat(appConfigPath); !os.IsNotExist(err) {
		t.Fatalf("expected no config file when domain empty, stat error = %v", err)
	}

	if err := manager.ApplyRouting(ctx, &domain.App{ID: "app-1", PublicDomain: "app.example.com", ProxyTargetPort: 0}, domain.PlatformSettings{}); err != nil {
		t.Fatalf("ApplyRouting() error = %v", err)
	}
	if _, err := os.Stat(appConfigPath); !os.IsNotExist(err) {
		t.Fatalf("expected no config file when proxy port is zero, stat error = %v", err)
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

func TestCertbotManager_RemoveCertificate_NoEntry_NoError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	manager := NewCertbotManagerWithBinary("certbot")
	manager.runScript = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "certbot" {
			t.Fatalf("expected certbot binary, got %q", name)
		}
		return []byte("No such entry \"app.example.com\""), errors.New("exit status 1")
	}

	if err := manager.RemoveCertificate(ctx, "app.example.com"); err != nil {
		t.Fatalf("RemoveCertificate() error = %v", err)
	}
}

func TestCertbotManager_EnsureCertificate_RequiresEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	manager := NewCertbotManagerWithBinary("certbot")
	err := manager.EnsureCertificate(ctx, domain.PlatformSettings{
		CertbotEnabled:       true,
		CertbotTermsAccepted: true,
	}, "app.example.com")
	if err == nil {
		t.Fatal("expected email required error")
	}
}

func TestCertbotManager_EnsureCertificate_RequiresTerms(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	manager := NewCertbotManagerWithBinary("certbot")
	err := manager.EnsureCertificate(ctx, domain.PlatformSettings{
		CertbotEnabled:  true,
		CertbotEmail:    "ops@example.com",
		CertbotStaging:  true,
	}, "app.example.com")
	if err == nil {
		t.Fatal("expected terms accepted error")
	}
}

