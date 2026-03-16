# GUI Docker

A lightweight Go-based PaaS panel for managing applications via Docker Compose.
Built with [templ](https://templ.guide) components and the shared `ui8kit` design system.

- **HTTP Basic Auth** with session-based login
- **BoltDB** storage for app metadata
- **Deploy / stop / restart / logs** via `docker compose` CLI
- **Web UI**: Overview, Apps, Compose editor, Logs, Scanner, Settings
- **Host nginx** routing with per-app reverse proxy configs
- **Host certbot** integration for Let's Encrypt TLS certificates

## Architecture

```text
(repo root)
├── ui8kit/                       # Shared headless UI component library (Go templ)
│   ├── ui/                       # Primitives: Box, Stack, Group, Button, Badge, Field...
│   ├── layout/                   # Shell, Sidebar, Header (full page layout)
│   ├── utils/                    # UtilityProps, cn(), variant functions
│   └── styles/                   # base.css, components.css, latty.css
├── localdeps/
│   └── templ/                    # Minimal templ runtime (no external dependency)
├── gui-docker/                   # Docker Compose admin panel
│   ├── cmd/                      # Application entry point
│   ├── config/                   # Environment-based configuration
│   ├── domain/                   # Entities, errors, port interfaces
│   ├── handlers/                 # HTTP page handlers + JSON API
│   ├── infrastructure/
│   │   ├── bolt/                 # App + settings repository (BoltDB)
│   │   ├── docker/               # Docker Compose CLI adapter
│   │   ├── git/                  # Git clone adapter
│   │   └── hosting/              # Nginx + Certbot host management
│   ├── middleware/                # Session auth
│   ├── usecase/
│   │   ├── app/                  # App lifecycle service
│   │   ├── scanner/              # Docker resource scanner
│   │   └── settings/             # Platform settings service
│   ├── pages/                    # Templ page components + view models
│   ├── views/                    # Renderer bridge (templ adapter)
│   ├── static/css/               # Tailwind source + compiled output
│   ├── data/                     # Legacy dashboard seed data
│   └── scripts/                  # Entrypoint, CSS build
```

## Server Requirements

### Ubuntu / Debian

```bash
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-plugin nginx certbot python3-certbot-nginx

sudo usermod -aG docker "$USER"
sudo mkdir -p /opt/stacks
sudo chown -R "$USER:$USER" /opt/stacks
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl restart nginx
```

After changing the `docker` group, re-login or run:

```bash
newgrp docker
```

## Clone and Deploy

### Option 1: Build and run on the host

```bash
git clone https://github.com/fastygo/guidocker.git
cd guidocker/gui-docker

# Пример для Go 1.22
wget https://go.dev/dl/go1.22.2.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version

go test ./...
go build -trimpath -ldflags="-s -w" -o gui-docker ./cmd

PAAS_ADMIN_USER=admin \
PAAS_ADMIN_PASS=admin@123 \
PAAS_PORT=3000 \
STACKS_DIR=/opt/stacks \
BOLT_DB_FILE=/opt/stacks/.paas.db \
./gui-docker
```

Then open: `http://localhost:3000`

### Option 2: Build release binary with Make

```bash
cd guidocker/gui-docker
make release
```

Copy `dist/gui-docker-linux-amd64` to the server and run with environment variables.

### Option 3: Docker container

Build from the repository root (the Dockerfile uses monorepo context):

```bash
git clone https://github.com/fastygo/guidocker.git
cd guidocker

docker build -f gui-docker/Dockerfile -t gui-docker:latest .
```

Or use Make from inside `gui-docker/`:

```bash
cd guidocker/gui-docker
make docker-build
```

Run the container:

```bash
docker network inspect paas-network >/dev/null 2>&1 || docker network create paas-network

docker run -d \
  --name gui-docker \
  --restart unless-stopped \
  --group-add "$(getent group docker | cut -d: -f3)" \
  --pid host \
  --network paas-network \
  -p 3000:3000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /:/host \
  -v /opt/stacks:/opt/stacks \
  -v /etc/nginx:/etc/nginx \
  -v /etc/letsencrypt:/etc/letsencrypt \
  -v /var/lib/letsencrypt:/var/lib/letsencrypt \
  -v /var/log/letsencrypt:/var/log/letsencrypt \
  --env SERVER_HOST=0.0.0.0 \
  --env PAAS_PORT=3000 \
  --env PAAS_ADMIN_USER=admin \
  --env PAAS_ADMIN_PASS=admin@123 \
  --env STACKS_DIR=/opt/stacks \
  --env BOLT_DB_FILE=/opt/stacks/.paas.db \
  --env PAAS_APP_NETWORK=paas-network \
  --env PAAS_HOST_ROOT=/host \
  --env PAAS_NGINX_BINARY=/usr/sbin/nginx \
  --env PAAS_NGINX_SITES_DIR=/etc/nginx/conf.d \
  --env PAAS_CERTBOT_BINARY=/usr/bin/certbot \
  gui-docker:latest
```

```bash
docker logs --tail 80 gui-docker
```

Update 

```bash
git pull origin main

git fetch origin current
git reset --hard origin/current

docker ps

docker stop gui-docker
docker rm gui-docker
```

Or use Make with auto GID detection:

```bash
cd guidocker/gui-docker
make docker-run-auto PAAS_ADMIN_USER=admin PAAS_ADMIN_PASS='admin@123'
```

## Development

### Prerequisites

```bash
cd guidocker/gui-docker
npm install
```

### CSS build

```bash
npm run build:css          # one-time build
npm run dev:css            # watch mode
```

Or with Make:

```bash
make css
```

### Dev mode (no Docker required)

```bash
cd guidocker/gui-docker
DASHBOARD_DEV_MODE=true DASHBOARD_AUTH_DISABLED=true go run ./cmd
```

Uses a mock Docker repository so the UI works without Docker installed.

### Quick checks

```bash
curl -u admin:admin@123 http://localhost:3000/api/apps
```

## Makefile Targets

Run all Make commands from inside `gui-docker/`:

```bash
make build           # Build local binary into ./bin
make release         # Build Linux amd64 release binary into ./dist
make run             # Build and run on the host
make dev             # Run with go run for development
make css             # Build Tailwind CSS
make test            # Run tests
make docker-build    # Build Docker image (context = repo root)
make docker-run      # Run container with all volume mounts
make docker-run-auto # Auto-detect host docker GID and run
make docker-stop     # Stop and remove container
make docker-logs     # Tail container logs
make docker-shell    # Open shell inside the running container
```

### Docker Runtime Requirements

- Mount host Docker socket: `-v /var/run/docker.sock:/var/run/docker.sock`
- Share the host PID namespace: `--pid host`
- Run on the managed Docker network: `--network paas-network`
- Mount the host root into the container: `-v /:/host`
- Mount stacks to `/opt/stacks` (writable by the container user)
- Mount host nginx and certbot state:
  `/etc/nginx`, `/etc/letsencrypt`, `/var/lib/letsencrypt`, `/var/log/letsencrypt`
- Keep host ports `80` and `443` free for host-installed `nginx`
- Create `paas-network` before starting the container
- Use `make docker-run-auto` on Linux for automatic host docker GID detection

### Post-start checklist

```bash
docker network inspect paas-network >/dev/null
docker inspect gui-docker --format '{{json .HostConfig.NetworkMode}}'
docker inspect gui-docker --format '{{json .HostConfig.PidMode}}'
docker exec gui-docker chroot /host /usr/sbin/nginx -t
docker exec gui-docker chroot /host /usr/bin/certbot --version
docker logs --tail 80 gui-docker
```

## Configuration

| Variable                  | Default                  | Description                                               |
| ------------------------- | ------------------------ | --------------------------------------------------------- |
| `SERVER_HOST`             | `localhost`              | HTTP server host                                          |
| `PAAS_PORT`               | `3000`                   | Main server port                                          |
| `PAAS_ADMIN_USER`         | `admin`                  | Login username                                            |
| `PAAS_ADMIN_PASS`         | `admin@123`              | Login password                                            |
| `DASHBOARD_AUTH_DISABLED`  | `false`                 | Disable auth for local dev                                |
| `DASHBOARD_DEV_MODE`      | `false`                  | Use mock Docker (UI testing without Docker)               |
| `PAAS_MOCK_UI`            | `false`                  | Alias for `DASHBOARD_DEV_MODE`                            |
| `STACKS_DIR`              | `/opt/stacks`            | Base directory for compose stacks                         |
| `BOLT_DB_FILE`            | `/opt/stacks/.paas.db`   | BoltDB file path                                          |
| `DASHBOARD_DATA_FILE`     | `data/dashboard.json`    | Legacy JSON dashboard source                              |
| `PAAS_APP_NETWORK`        | `paas-network`           | Managed Docker network for routed applications            |
| `PAAS_HOST_ROOT`          | `/host`                  | Mounted host root used to run host nginx/certbot commands |
| `PAAS_NGINX_BINARY`       | `/usr/sbin/nginx`        | Host nginx binary path used through `chroot`              |
| `PAAS_NGINX_SITES_DIR`    | `/etc/nginx/conf.d`      | Host nginx directory where route configs are written      |
| `PAAS_CERTBOT_BINARY`     | `/usr/bin/certbot`       | Host certbot binary path used through `chroot`            |

## Routes

### Web UI

| Path                  | Description    |
| --------------------- | -------------- |
| `GET /`               | Overview       |
| `GET /login`          | Login screen   |
| `GET /apps`           | App list       |
| `GET /apps/new`       | Create app     |
| `GET /apps/{id}`      | App detail     |
| `GET /apps/{id}/compose` | Compose editor |
| `GET /apps/{id}/logs` | Logs viewer    |
| `GET /scan`           | Scanner UI     |
| `GET /settings`       | Settings       |

### API

| Method | Path                          | Description    |
| ------ | ----------------------------- | -------------- |
| GET    | `/api/dashboard`              | Dashboard data |
| GET    | `/api/apps`                   | List apps      |
| POST   | `/api/apps`                   | Create app     |
| POST   | `/api/apps/import`            | Import from Git |
| GET    | `/api/apps/{id}`              | App detail     |
| PUT    | `/api/apps/{id}`              | Update app     |
| DELETE | `/api/apps/{id}`              | Delete app     |
| POST   | `/api/apps/{id}/deploy`       | Deploy         |
| POST   | `/api/apps/{id}/stop`         | Stop           |
| POST   | `/api/apps/{id}/restart`      | Restart        |
| GET    | `/api/apps/{id}/logs?lines=N` | Logs           |
| GET    | `/api/apps/{id}/config`       | Get app config |
| PUT    | `/api/apps/{id}/config`       | Update config  |
| GET    | `/api/settings`               | Get settings   |
| PUT    | `/api/settings`               | Update settings|
| POST   | `/api/certificates/renew`     | Renew certs    |
| GET    | `/api/scan`                   | Scanner results|

### API Examples

```bash
curl -u admin:admin@123 http://localhost:3000/api/apps

curl -u admin:admin@123 \
  -X POST http://localhost:3000/api/apps \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo-nginx",
    "compose_yaml": "services:\n  web:\n    image: nginx:alpine\n    expose:\n      - \"80\""
  }'

curl -u admin:admin@123 \
  -X POST http://localhost:3000/api/apps/demo-id/deploy
```

## Production Setup

### systemd unit

```ini
# /etc/systemd/system/gui-docker.service
[Unit]
Description=GUI Docker Panel
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/opt/gui-docker
Environment=PAAS_ADMIN_USER=admin
Environment=PAAS_ADMIN_PASS=admin@123
Environment=PAAS_PORT=3000
Environment=STACKS_DIR=/opt/stacks
Environment=BOLT_DB_FILE=/opt/stacks/.paas.db
ExecStart=/opt/gui-docker/gui-docker
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Apply:

```bash
sudo systemctl daemon-reload
sudo systemctl enable gui-docker
sudo systemctl start gui-docker
sudo systemctl status gui-docker
```

### Host nginx and certbot

The panel does not ship its own ingress runtime.
Install `nginx` and `certbot` on the host and let the panel manage:

- per-app route files in `/etc/nginx/conf.d`
- certificate issuance and removal through host `certbot`
- app-to-app isolation through the external Docker network `paas-network`

After changing host nginx configuration outside the panel, always verify:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

### HTTPS and certificate automation

Use this sequence after the app is already reachable over plain HTTP on its domain.

**Admin settings**

- `Enable certificate automation` -- turn on host `certbot` integration.
- `Use Let's Encrypt staging environment` -- use only for test issuance.
- `Enable automatic renewal` -- keep enabled for normal operation.
- `I accept Let's Encrypt terms of service` -- required before certificate issuance.
- `Save admin settings` -- save platform TLS settings before enabling HTTPS on a specific app.
- `Run certificate renewal now` -- manual renewal or verification action.

**App settings**

1. Set `PublicDomain`.
2. Set `ProxyTargetPort` to the internal container port.
3. Save the app.
4. Enable `Enable HTTPS on proxy`.
5. Save the app again to trigger certificate issuance and HTTPS routing.

**Verification commands**

```bash
ls -la /etc/letsencrypt/live/<domain>
certbot certificates
nginx -T | sed -n '/<domain>/,/}/p'
curl -I https://<domain>/
echo | openssl s_client -connect <domain>:443 -servername <domain> 2>/dev/null | \
  openssl x509 -noout -subject -issuer -dates
```

**Renewal checks**

```bash
certbot renew --dry-run
docker logs --tail 100 gui-docker
nginx -t
```

**Switching from staging to production**

1. Disable `Use Let's Encrypt staging environment` in admin settings and save.
2. Open the app, disable `Enable HTTPS on proxy`, save.
3. Re-enable `Enable HTTPS on proxy`, save again.
4. Verify the issuer no longer contains `(STAGING)`.

## Tests

```bash
cd guidocker
go test ./...
```

Covered areas: config loading, app lifecycle, BoltDB CRUD, handlers, scanner.

## Current Limitations

- One routed upstream per app
- Routing uses resolved container IPs; if an app gets a new IP while the panel is absent, restart the panel to refresh the route
- No multi-user / RBAC
- No wildcard SSL
