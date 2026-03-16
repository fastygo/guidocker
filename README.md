# Smart GUI Control Plane

A lightweight PaaS for managing Docker applications. Deploy, monitor, and clean up compose-based stacks from a single control surface.

## Architecture

```text
@GUIDocker/
├── ui8kit/              # Shared headless UI component library (Go templ)
│   ├── ui/              # Primitives: Box, Stack, Group, Button, Badge, Field...
│   ├── layout/          # Shell, Sidebar, Header (full page layout)
│   ├── utils/           # UtilityProps, cn(), variant functions
│   └── styles/          # base.css, components.css, latty.css
│
├── gui-docker/          # Go application — main PaaS server
│   ├── cmd/             # Application entry point
│   ├── config/          # Configuration
│   ├── domain/          # Entities, ports
│   ├── infrastructure/  # BoltDB, Docker CLI, Git, Nginx, Certbot
│   ├── handlers/        # HTTP page handlers, REST API
│   ├── usecase/         # App lifecycle, scanner, settings
│   ├── pages/           # Templ page components
│   └── static/          # Tailwind CSS, assets
│
├── website/             # Public home page / landing site
│   ├── docker-compose.yml  # Importable static site stack
│   ├── index.html       # Product intro, feature highlights
│   ├── style.input.css  # Tailwind source
│   └── style.css        # Compiled CSS (build output)
│
└── .project/            # Project planning notes and internal docs
```

- **ui8kit/** — Headless component library built with [templ](https://templ.guide). Reusable across gui-docker and future admin panels (gui-php, gui-bare-sections, etc.).
- **gui-docker/** — Go server with HTTP Basic Auth, BoltDB, and Docker Compose integration. Serves the full admin UI (Overview, Apps, Compose, Logs, Scanner, Settings) and REST API.
- **website/** — Static landing page (HTML + Tailwind) for the public product home page. Built via `npm run build:www` from `gui-docker/` and importable into the admin panel via `website/docker-compose.yml`.

## Quick Start

```bash
cd gui-docker
go build -trimpath -ldflags="-s -w" -o gui-docker ./cmd
PAAS_ADMIN_USER=admin PAAS_ADMIN_PASS=admin@123 ./gui-docker
```

Open `http://localhost:3000`.

## Build Frontend Assets

```bash
cd gui-docker
npm install

# Admin panel UI (Tailwind)
npm run build:css

# Landing page (website/)
npm run build:www
```

## Deployment Model

- The panel remains an autonomous temporary GUI for a single root operator.
- Applications are deployed by Docker Compose and continue running after the panel container is removed.
- Public routing is handled by host-installed `nginx` and `certbot`, managed by the panel.
- Applications are attached to the managed Docker network `paas-network` and should be routed by internal container port, not by published host ports.

## Website Import Checklist

Use this flow to deploy the official landing page through the admin panel:

1. Import from Git.
2. Use repository URL `https://github.com/fastygo/guidocker.git`.
3. Set compose file path to `website/docker-compose.yml`.
4. Leave app port empty because the compose file already defines the service.
5. Deploy the imported app.
6. Open app settings in the admin panel.
7. Set `ProxyTargetPort` to `80` because routing now targets the internal container port on `paas-network`.
8. Save the app configuration.
9. Add the public domain later through routing/settings in the admin panel.
10. Enable HTTPS only after DNS is ready and host certbot settings are configured.

## HTTPS And Certificate Automation

Use this sequence after the site is already reachable over plain HTTP on its domain.

### Admin settings

- `Enable certificate automation`
  Turn on host `certbot` integration for the platform.
- `Use Let's Encrypt staging environment`
  Use this only for test issuance. Staging certificates are expected to be untrusted by browsers and `curl`.
- `Enable automatic renewal`
  Keep this enabled for normal operation. Disable it only if you plan to run renewal manually on the host.
- `I accept Let's Encrypt terms of service`
  Required before the panel can issue a certificate.
- `Save admin settings`
  Save platform TLS settings before enabling HTTPS on a specific app.
- `Run certificate renewal now`
  Use this as a manual renewal/verification action. It is not required during the initial certificate issuance flow.

### App settings

1. Set `PublicDomain`.
2. Set `ProxyTargetPort` to the internal container port.
   For `website/docker-compose.yml` this is `80`.
3. Save the app.
4. Enable `Enable HTTPS on proxy`.
5. Save the app again to trigger certificate issuance and HTTPS routing.

### Verification Commands

Check that the certificate exists on disk:

```bash
ls -la /etc/letsencrypt/live/<domain>
```

Check what `certbot` knows about the certificate:

```bash
certbot certificates
```

Check the effective nginx config for the routed domain:

```bash
nginx -T | sed -n '/<domain>/,/}/p'
```

Check HTTPS from the server:

```bash
curl -I https://<domain>/
```

Inspect issuer and validity dates:

```bash
echo | openssl s_client -connect <domain>:443 -servername <domain> 2>/dev/null | \
  openssl x509 -noout -subject -issuer -dates
```

### Expected Results

- If staging is enabled, the certificate issuer will include `(STAGING)` and clients will not trust it.
- If staging is disabled, the issuer should be a normal Let's Encrypt production CA and `curl -I https://<domain>/` should succeed without certificate errors.
- `nginx -T` should show `listen 443 ssl;` and certificate paths under `/etc/letsencrypt/live/<domain>/`.

### Renewal Checks

Use the dashboard button `Run certificate renewal now` when you want to verify the renewal path or renew near expiry.
Equivalent terminal checks:

```bash
certbot renew --dry-run
docker logs --tail 100 gui-docker
nginx -t
```

Expected result:

- `certbot renew --dry-run` succeeds
- panel logs do not show certificate errors
- `nginx -t` remains successful after renewal

### Switching From Staging To Production

1. Disable `Use Let's Encrypt staging environment` in admin settings.
2. Save admin settings.
3. Open the app.
4. Disable `Enable HTTPS on proxy` and save.
5. Re-enable `Enable HTTPS on proxy` and save again.
6. Re-run the verification commands above and confirm the issuer no longer contains `(STAGING)`.

## Documentation

- [gui-docker/README.md](gui-docker/README.md) — Full panel docs: setup, API, deployment, Makefile targets, nginx/certbot.
