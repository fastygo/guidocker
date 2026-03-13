# PaaS Dashboard MVP

A lightweight Go-based PaaS service for managing applications via Docker Compose:

- **HTTP Basic Auth** with session-based login
- **BoltDB** storage for app metadata
- **Deploy / stop / restart / logs** via `docker compose` CLI
- **Web UI**: Overview, Apps, Compose editor, Logs, Scanner, Settings
- Fallback to legacy JSON dashboard for compatibility

## Architecture

```text
dashboard/
├── config/                    # Environment-based configuration
├── domain/                    # Entities, errors, port interfaces
├── infrastructure/
│   ├── bolt/                  # App repository (BoltDB)
│   ├── docker/                # Docker CLI adapter
│   └── repository.go         # Legacy JSON dashboard repository
├── interfaces/
│   ├── middleware/            # Basic Auth + session login
│   ├── handlers.go            # Legacy dashboard handlers
│   ├── paas_handlers.go       # PaaS pages + REST API
│   └── scan_handlers.go       # Scanner UI + API
├── usecase/
│   ├── app/                   # App lifecycle service
│   └── scanner/               # Docker resource scanner
├── views/                     # HTML templates
├── static/                    # CSS (Tailwind), assets
├── data/                      # Legacy dashboard seed data
└── main.go                    # Server wiring
```

## Features

- Create apps from `docker-compose.yml` YAML
- Store metadata in BoltDB
- Persist compose stacks to `STACKS_DIR/<app-id>/docker-compose.yml`
- Run `docker compose up -d`, `down`, `restart`
- Stream logs via API
- Display app status in the UI
- Scan Docker resources and reconcile with stored apps

## Server Requirements

### Ubuntu / Debian

```bash
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-plugin nginx

# If docker compose plugin is unavailable in the repo:
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
  -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

sudo usermod -aG docker "$USER"
sudo mkdir -p /opt/stacks
sudo chown -R "$USER:$USER" /opt/stacks
```

After changing the `docker` group, re-login or run:

```bash
newgrp docker
```

## Build and Run

```bash
cd dashboard
go test ./...
go build -o dashboard .

PAAS_ADMIN_USER=admin \
PAAS_ADMIN_PASS=admin@123 \
PAAS_PORT=3000 \
STACKS_DIR=/opt/stacks \
BOLT_DB_FILE=/opt/stacks/.paas.db \
./dashboard
```

Then:

- **UI**: `http://localhost:3000`
- **Login**: `http://localhost:3000/login`

## Development (Tailwind CLI + Air)

Prerequisites:

```bash
cd dashboard
npm install
go install github.com/air-verse/air@latest
```

`dashboard/package.json` scripts:

| Script      | Description                          |
| ----------- | ------------------------------------ |
| `dev:css`   | Tailwind CSS watch mode              |
| `build:css` | One-time CSS build                   |
| `build:www` | Build landing page CSS (`.project/www`) |
| `dev:air`   | Runs `dev:css` + `air` together      |

**Two-terminal flow:**

- **Terminal 1:** `npm run build:css`
- **Terminal 2:** `air`

Or use a single terminal: `npm run dev:air`.

**Disable auth for local iteration:**

```bash
cd dashboard
DASHBOARD_AUTH_DISABLED=true air
```

**Quick checks:**

- Open: `http://localhost:3000/apps` and `http://localhost:3000/apps/new`
- API: `curl -u admin:admin@123 http://localhost:3000/api/apps`

## Makefile Targets

```bash
# Build local binary into ./bin
make build

# Build Linux release binary into ./dist
make release

# Run tests
make test

# Build Docker image
make docker-build IMAGE_NAME=paas-dashboard IMAGE_TAG=latest

# Run container
make docker-run IMAGE_NAME=paas-dashboard IMAGE_TAG=latest \
  CONTAINER_NAME=paas-dashboard \
  PAAS_ADMIN_USER=admin \
  PAAS_ADMIN_PASS='admin@123' \
  STACKS_DIR=/opt/stacks
```

### Docker Runtime Requirements

- Mount host Docker socket: `-v /var/run/docker.sock:/var/run/docker.sock`
- Add `--group-add <host-docker-GID>` so the container user can access the socket
- Mount stacks to `/opt/stacks` (writable by the container user)
- Use `make docker-run-auto` on Linux for automatic host docker GID detection

**Example:**

```bash
docker run -d \
  --name paas-dashboard \
  --restart unless-stopped \
  --group-add 999 \
  -p 3000:3000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /opt/stacks:/opt/stacks \
  --env PAAS_ADMIN_USER=admin \
  --env PAAS_ADMIN_PASS=admin@123 \
  --env STACKS_DIR=/opt/stacks \
  paas-dashboard:latest
```

**Ensure `/opt/stacks` ownership:**

```bash
HOST_PAAS_UID=$(docker run --rm --entrypoint "" paas-dashboard:latest id -u paas)
sudo chown -R "$HOST_PAAS_UID":"$HOST_PAAS_UID" /opt/stacks
```

## Configuration

| Variable               | Default                 | Description                          |
| ---------------------- | ----------------------- | ------------------------------------ |
| `SERVER_HOST`          | `localhost`             | HTTP server host                     |
| `PAAS_PORT`            | `3000`                  | Main server port                     |
| `SERVER_PORT`          | `3000`                  | Legacy fallback port                 |
| `PAAS_ADMIN_USER`      | `admin`                 | Login username                       |
| `PAAS_ADMIN_PASS`      | `admin@123`             | Login password                       |
| `DASHBOARD_AUTH_DISABLED` | `false`             | Disable auth for local dev           |
| `STACKS_DIR`           | `/opt/stacks`           | Base directory for compose stacks    |
| `BOLT_DB_FILE`         | `/opt/stacks/.paas.db`  | BoltDB file path                     |
| `DASHBOARD_DATA_FILE`  | `data/dashboard.json`    | Legacy JSON dashboard source         |

## Routes

### Web UI

| Path              | Description              |
| ----------------- | ------------------------ |
| `GET /`           | Overview                 |
| `GET /login`      | Login screen             |
| `GET /apps`       | App list                 |
| `GET /apps/new`   | Create app               |
| `GET /apps/{id}`  | App detail               |
| `GET /apps/{id}/compose` | Compose editor   |
| `GET /apps/{id}/logs`    | Logs viewer      |
| `GET /scan`       | Scanner UI                |
| `GET /settings`   | Settings                 |

### API

| Method | Path                      | Description        |
| ------ | ------------------------- | ------------------ |
| GET    | `/api/dashboard`          | Dashboard data     |
| GET    | `/api/apps`               | List apps          |
| POST   | `/api/apps`               | Create app         |
| GET    | `/api/apps/{id}`          | App detail         |
| PUT    | `/api/apps/{id}`          | Update app         |
| DELETE | `/api/apps/{id}`          | Delete app         |
| POST   | `/api/apps/{id}/deploy`   | Deploy             |
| POST   | `/api/apps/{id}/stop`     | Stop               |
| POST   | `/api/apps/{id}/restart`  | Restart            |
| GET    | `/api/apps/{id}/logs?lines=100` | Logs stream  |
| GET    | `/api/scan`               | Scanner results    |

### API Examples

```bash
curl -u admin:admin@123 http://localhost:3000/api/apps

curl -u admin:admin@123 \
  -X POST http://localhost:3000/api/apps \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo-nginx",
    "compose_yaml": "version: \"3.9\"\nservices:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\""
  }'

curl -u admin:admin@123 \
  -X POST http://localhost:3000/api/apps/demo-id/deploy
```

## Production Setup

### systemd unit

```ini
# /etc/systemd/system/paas.service
[Unit]
Description=PaaS Dashboard
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/opt/paas
Environment=PAAS_ADMIN_USER=admin
Environment=PAAS_ADMIN_PASS=admin@123
Environment=PAAS_PORT=3000
Environment=STACKS_DIR=/opt/stacks
Environment=BOLT_DB_FILE=/opt/stacks/.paas.db
ExecStart=/opt/paas/dashboard
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Apply:

```bash
sudo systemctl daemon-reload
sudo systemctl enable paas
sudo systemctl start paas
sudo systemctl status paas
```

### nginx reverse proxy

```nginx
# /etc/nginx/sites-enabled/paas
server {
    listen 80;
    server_name _;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Apply:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## Tests

```bash
go test ./...
```

Covered areas:

- Config loading
- App use case lifecycle
- Bolt repository CRUD
- Handlers: login, create app, deploy

## MVP Limitations

- No automatic nginx config generation per app
- No wildcard SSL / Let's Encrypt
- No multi-user / RBAC
- No background sync of container state with stored apps
