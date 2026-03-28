# Dashboard

This directory contains the Go control plane used to manage Docker applications, host `nginx` routing, and host `certbot` operations.

## Runtime model

- internal-only host endpoint: `http://127.0.0.1:7000`
- default hardened mode: `DASHBOARD_MODE=api`
- optional GUI mode for operator sessions through SSH tunneling
- persistent state lives outside the container in `/opt/stacks`
- host `nginx` and `certbot` continue serving managed apps even if the dashboard is reinstalled

## Build and run locally

```bash
cd dashboard
go test ./...
go build -o dashboard .

PAAS_ADMIN_USER=admin \
PAAS_ADMIN_PASS=admin@123 \
PAAS_PORT=7000 \
STACKS_DIR=/opt/stacks \
BOLT_DB_FILE=/opt/stacks/.paas.db \
./dashboard
```

Open:

- UI: `http://localhost:7000`
- Login: `http://localhost:7000/login`

## Development helpers

Prerequisites:

```bash
cd dashboard
npm install
go install github.com/air-verse/air@latest
```

Common loops:

```bash
npm run build:css
air
```

Disable auth for UI iteration:

```bash
DASHBOARD_AUTH_DISABLED=true air
```

Quick checks:

- `http://localhost:7000/apps`
- `http://localhost:7000/apps/new`
- `curl -u admin:admin@123 http://localhost:7000/api/apps`

## Docker runtime

The root-level `Dockerfile` and `docker-compose.yml` are the supported runtime artifacts.

Important characteristics:

- published only as `127.0.0.1:7000:7000`
- container listens on `0.0.0.0:7000` internally
- root filesystem is read-only
- writable state is externalized to mounted host paths
- default container mode is `DASHBOARD_MODE=api`

Current writable paths:

- `/opt/stacks`
- `/etc/nginx`
- `/etc/letsencrypt`
- `/var/lib/letsencrypt`
- `/var/log/letsencrypt`
- `/tmp` via `tmpfs`

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `SERVER_HOST` | `localhost` | HTTP server host for local binary runs |
| `PAAS_PORT` | `7000` | Controller port |
| `SERVER_PORT` | `7000` | Legacy alias for controller port |
| `PAAS_ADMIN_USER` | `admin` | Login username |
| `PAAS_ADMIN_PASS` | `admin@123` | Login password |
| `DASHBOARD_AUTH_DISABLED` | `false` | Disable auth for local dev only |
| `STACKS_DIR` | `/opt/stacks` | Base directory for compose stacks and runtime state |
| `BOLT_DB_FILE` | `/opt/stacks/.paas.db` | BoltDB file path |
| `DASHBOARD_DATA_FILE` | `data/dashboard.json` | Legacy JSON data source for local UI development |
| `PAAS_APP_NETWORK` | `paas-network` | Managed Docker network for routed applications |
| `PAAS_HOST_ROOT` | `/host` | Mounted host root used to run host binaries through `chroot` |
| `PAAS_NGINX_BINARY` | `/usr/sbin/nginx` | Host nginx binary path |
| `PAAS_NGINX_SITES_DIR` | `/etc/nginx/conf.d` | Host nginx config directory for managed routes |
| `PAAS_CERTBOT_BINARY` | `/usr/bin/certbot` | Host certbot binary path |
| `DASHBOARD_MODE` | `gui` | `gui` for operator sessions, `api` for hardened runtime |

## Routes

### Web UI

| Path | Description |
| --- | --- |
| `GET /` | Overview |
| `GET /login` | Login screen |
| `GET /apps` | App list |
| `GET /apps/new` | Create app |
| `GET /apps/{id}` | App detail |
| `GET /apps/{id}/compose` | Compose editor |
| `GET /apps/{id}/logs` | Logs viewer |
| `GET /scan` | Scanner UI |
| `GET /settings` | Platform TLS settings |

### API

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/dashboard` | Dashboard data |
| GET | `/api/apps` | List apps |
| POST | `/api/apps` | Create app |
| GET | `/api/apps/{id}` | App detail |
| PUT | `/api/apps/{id}` | Update app |
| DELETE | `/api/apps/{id}` | Delete app |
| POST | `/api/apps/{id}/deploy` | Deploy |
| POST | `/api/apps/{id}/stop` | Stop |
| POST | `/api/apps/{id}/restart` | Restart |
| GET | `/api/apps/{id}/logs?lines=100` | Logs stream |
| GET | `/api/settings` | Platform TLS settings |
| PUT | `/api/settings` | Update platform TLS settings |
| POST | `/api/certificates/renew` | Run certbot renew |
| GET | `/api/scan` | Scanner results |
| GET | `/api/health` | Health probe |

## Internal-only operations

Recommended production access:

```bash
ssh -L 7500:127.0.0.1:7000 root@<SERVER_HOST>
```

Then:

```text
http://127.0.0.1:7500
```

If the runtime stays in `DASHBOARD_MODE=api`, the tunnel still works but only `/api/*` routes are available.

## Platform TLS settings

The settings page now manages only TLS automation for managed application domains.

Workflow:

1. Save platform certbot settings in `/settings`.
2. Configure an app `PublicDomain` and `ProxyTargetPort`.
3. Enable `Enable HTTPS on proxy` for that app only after plain HTTP is reachable.

Useful server checks:

```bash
sudo nginx -t
sudo systemctl reload nginx
certbot certificates
curl -I https://<domain>/
```

## Tests

```bash
go test ./...
```
