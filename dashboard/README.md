# PaaS Dashboard MVP

Лёгкий PaaS-сервис на Go для управления приложениями через Docker Compose:

- HTTP Basic Auth + login screen
- BoltDB-хранилище метаданных приложений
- deploy/stop/restart/logs через `docker compose`
- браузерные экраны: Overview, Apps, Compose, Logs
- fallback на старый JSON dashboard для совместимости

## Архитектура

```text
dashboard/
├── config/                    # ENV configuration
├── domain/                    # Entities, errors, interfaces
├── infrastructure/
│   ├── bolt/                  # App repository over BoltDB
│   ├── docker/                # Docker CLI adapter
│   └── repository.go          # Legacy JSON dashboard repository
├── interfaces/
│   ├── middleware/            # Basic Auth + session login
│   ├── handlers.go            # Legacy dashboard handlers
│   └── paas_handlers.go       # PaaS pages + API
├── usecase/app/               # App lifecycle service
├── data/                      # Legacy dashboard seed data
└── main.go                    # Server wiring
```

## Что умеет MVP

- создавать приложение по `docker-compose.yml`
- хранить метаданные в BoltDB
- сохранять compose-стек в `STACKS_DIR/<app-id>/docker-compose.yml`
- запускать `docker compose up -d`, `down`, `restart`
- отдавать логи через API
- отображать статус приложения в UI

## Требования к серверу

### Ubuntu / Debian setup

```bash
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-plugin nginx

# Если docker compose plugin недоступен в репозитории:
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
  -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

sudo usermod -aG docker "$USER"
sudo mkdir -p /opt/stacks
sudo chown -R "$USER:$USER" /opt/stacks
```

После изменения группы `docker` перелогиньтесь или выполните:

```bash
newgrp docker
```

## Сборка и локальный запуск

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

После запуска:

- UI: `http://localhost:3000`
- Login: `http://localhost:3000/login`

Для живой разработки UI через `air` можно отключить авторизацию так:

```bash
cd dashboard
DASHBOARD_AUTH_DISABLED=true npm run dev:air
```

## Makefile targets

Основные команды для VPS:

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

## Конфигурация

### Основные переменные окружения

| Variable | Default | Description |
| --- | --- | --- |
| `SERVER_HOST` | `localhost` | Host for HTTP server |
| `PAAS_PORT` | `3000` | Main server port |
| `SERVER_PORT` | `3000` | Legacy fallback port variable |
| `PAAS_ADMIN_USER` | `admin` | Login username |
| `PAAS_ADMIN_PASS` | `admin@123` | Login password |
| `DASHBOARD_AUTH_DISABLED` | `false` | Disable auth middleware for local dev |
| `STACKS_DIR` | `/opt/stacks` | Base directory for compose stacks |
| `BOLT_DB_FILE` | `/opt/stacks/.paas.db` | BoltDB file |
| `DASHBOARD_DATA_FILE` | `data/dashboard.json` | Legacy JSON dashboard source |

## Основные роуты

### Web UI

- `GET /` — overview
- `GET /login` — login screen
- `GET /apps` — список приложений
- `GET /apps/new` — создание приложения
- `GET /apps/{id}` — карточка приложения
- `GET /apps/{id}/compose` — редактор compose
- `GET /apps/{id}/logs` — просмотр логов

### API

- `GET /api/dashboard`
- `GET /api/apps`
- `POST /api/apps`
- `GET /api/apps/{id}`
- `PUT /api/apps/{id}`
- `DELETE /api/apps/{id}`
- `POST /api/apps/{id}/deploy`
- `POST /api/apps/{id}/stop`
- `POST /api/apps/{id}/restart`
- `GET /api/apps/{id}/logs?lines=100`

### Пример API

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

Применить:

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

Применить:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## Тесты

```bash
go test ./...
```

Покрыты ключевые сценарии:

- config loading
- app use case lifecycle
- Bolt repository CRUD
- handlers: login, create app, deploy

## Ограничения MVP

- без автоматической генерации nginx-конфигов под каждое приложение
- без wildcard SSL / Let's Encrypt
- без multi-user / RBAC
- без фоновой синхронизации состояния контейнеров
