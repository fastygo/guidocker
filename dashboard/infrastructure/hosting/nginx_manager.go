package hosting

import (
	"context"
	"dashboard/domain"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultNginxSitesDir = "/etc/nginx/conf.d"
const defaultNginxBinary = "nginx"
const defaultProxyUpstreamHost = "host.docker.internal"
const managedConfigNameTemplate = "paas-app-%s.conf"
var certBasePath = "/etc/letsencrypt/live"

type NginxHostManager struct {
	binary    string
	sitesDir  string
	host      string
	runScript func(context.Context, string, ...string) ([]byte, error)
}

func NewNginxHostManager() *NginxHostManager {
	binary := strings.TrimSpace(os.Getenv("PAAS_NGINX_BINARY"))
	sitesDir := strings.TrimSpace(os.Getenv("PAAS_NGINX_SITES_DIR"))
	upstreamHost := strings.TrimSpace(os.Getenv("PAAS_PROXY_UPSTREAM_HOST"))
	return NewNginxHostManagerWithOptions(binary, sitesDir, upstreamHost)
}

func NewNginxHostManagerWithOptions(binary, sitesDir, upstreamHost string) *NginxHostManager {
	trimmedBinary := strings.TrimSpace(binary)
	if trimmedBinary == "" {
		trimmedBinary = defaultNginxBinary
	}
	trimmedSitesDir := strings.TrimSpace(sitesDir)
	if trimmedSitesDir == "" {
		trimmedSitesDir = defaultNginxSitesDir
	}
	trimmedHost := strings.TrimSpace(upstreamHost)
	if trimmedHost == "" {
		trimmedHost = defaultProxyUpstreamHost
	}
	return &NginxHostManager{
		binary:    trimmedBinary,
		sitesDir:  trimmedSitesDir,
		host:      trimmedHost,
		runScript: runCommand,
	}
}

func (m *NginxHostManager) ApplyRouting(_ context.Context, app *domain.App, _ domain.PlatformSettings) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	domainValue := strings.TrimSpace(app.PublicDomain)
	if domainValue == "" || app.ProxyTargetPort <= 0 {
		return nil
	}
	if err := os.MkdirAll(m.sitesDir, 0o755); err != nil {
		return fmt.Errorf("create nginx sites directory: %w", err)
	}

	config, err := buildNginxConfig(*app, m.host)
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath(app), []byte(config), 0o644)
}

func (m *NginxHostManager) RemoveRouting(_ context.Context, app *domain.App, _ domain.PlatformSettings) error {
	if app == nil {
		return nil
	}
	path := m.configPath(app)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove nginx route config: %w", err)
	}
	return nil
}

func (m *NginxHostManager) ValidateRouting(ctx context.Context) error {
	_, err := m.runScript(ctx, m.binary, "-t")
	if err != nil {
		return fmt.Errorf("validate nginx config: %w", err)
	}
	return nil
}

func (m *NginxHostManager) ReloadRouting(ctx context.Context) error {
	_, err := m.runScript(ctx, m.binary, "-s", "reload")
	if err != nil {
		return fmt.Errorf("reload nginx: %w", err)
	}
	return nil
}

func (m *NginxHostManager) configPath(app *domain.App) string {
	appID := strings.TrimSpace(app.ID)
	if appID == "" {
		appID = sanitizeFileName(strings.TrimSpace(app.PublicDomain))
	}
	return filepath.Join(m.sitesDir, fmt.Sprintf(managedConfigNameTemplate, appID))
}

func buildNginxConfig(app domain.App, proxyHost string) (string, error) {
	domainValue := strings.TrimSpace(app.PublicDomain)
	if domainValue == "" {
		return "", domain.ErrInvalidDomain
	}
	if app.ProxyTargetPort <= 0 || app.ProxyTargetPort > 65535 {
		return "", domain.ErrInvalidProxyPort
	}

	targetPort := app.ProxyTargetPort
	target := fmt.Sprintf("http://%s:%d", normalizeUpstreamHost(proxyHost), targetPort)
	useTLS := app.UseTLS
	certInfo := certificateFiles(app.PublicDomain)
	haveCert := certInfoExists(certInfo.fullChain, certInfo.privateKey)
	httpBlock := `
server {
    listen 80;
    server_name ` + domainValue + `;
`
	if useTLS && haveCert {
		httpBlock += `
    return 301 https://$host$request_uri;
}`
	} else {
		httpBlock += `
    location / {
        proxy_pass ` + target + `;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}`
	}

	var tlsBlock string
	if useTLS && haveCert {
		tlsBlock = `
server {
    listen 443 ssl;
    server_name ` + app.PublicDomain + `;
    ssl_certificate ` + certInfo.fullChain + `;
    ssl_certificate_key ` + certInfo.privateKey + `;
    location / {
        proxy_pass ` + target + `;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}`
	}

	return httpBlock + "\n" + tlsBlock, nil
}

func normalizeUpstreamHost(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultProxyUpstreamHost
	}
	return trimmed
}

type certFiles struct {
	fullChain string
	privateKey string
}

func certificateFiles(domainName string) certFiles {
	base := filepath.Join(certBasePath, strings.TrimSpace(domainName))
	return certFiles{
		fullChain: filepath.Join(base, "fullchain.pem"),
		privateKey: filepath.Join(base, "privkey.pem"),
	}
}

func certInfoExists(fullChainPath, privateKeyPath string) bool {
	if _, err := os.Stat(fullChainPath); err != nil {
		return false
	}
	if _, err := os.Stat(privateKeyPath); err != nil {
		return false
	}
	return true
}

func sanitizeFileName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "app"
	}
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, value)
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}


