# Internal Dashboard and App Deployment Guide

This `.paas` folder manages two related workflows:

- the dashboard itself as an internal-only controller
- regular applications deployed through that dashboard API

Related operator documentation:

- `docs/architecture/overview.md`
- `docs/deployment/dashboard-runtime.md`
- `docs/operations/settings-and-persistence.md`
- `docs/runbooks/dashboard-operations.md`
- `docs/runbooks/managed-app-operations.md`
- `docs/faq/operator-faq.md`

Target runtime model:

- dashboard API and optional GUI are available only on the server at `http://127.0.0.1:7000`
- public ingress belongs only to managed applications
- dashboard installation and updates happen directly through Docker on the server
- the dashboard never deploys itself through its own API

## Supported flows

### Dashboard runtime extensions

| Extension | Purpose |
| --- | --- |
| `bootstrap-direct` | First install on a clean server or a full reinstall |
| `deploy-direct` | Update the dashboard by building the image on the server |
| `deploy` | Update the dashboard through a registry-backed image |

### App deployment aliases

| Extension | Purpose |
| --- | --- |
| `app-bootstrap-direct` | Build the current repo on the server, create a new app through the dashboard API, and deploy it |
| `app-deploy-direct` | Build the current repo on the server, update an existing app through the dashboard API, and deploy it |
| `app-deploy` | Build and push the current repo through a registry, update an existing app through the dashboard API, and deploy it |

## Required inputs

### Shared controller inputs

| Input | Meaning |
| --- | --- |
| `INPUT_DASHBOARD_URL` | Internal dashboard URL, normally `http://127.0.0.1:7000` |
| `INPUT_DASHBOARD_USER` | Dashboard login user |
| `INPUT_DASHBOARD_PASS` | Dashboard login password |
| `INPUT_REGISTRY_HOST` | Registry host for registry-backed flows |
| `INPUT_IMAGE_REPOSITORY` | Registry repository for registry-backed flows |
| `INPUT_REGISTRY_USERNAME` | Registry username for registry-backed flows |
| `INPUT_REGISTRY_PASSWORD` | Registry password for registry-backed flows |

### Dashboard runtime inputs

| Input | Meaning |
| --- | --- |
| `INPUT_APP_NAME` | Docker Compose project / runtime directory name, usually `paas-dashboard` |
| `INPUT_CERTBOT_EMAIL` | Optional platform TLS email to seed during `bootstrap-direct` |
| `INPUT_CERTBOT_STAGING` | Optional Let's Encrypt staging flag |
| `INPUT_CERTBOT_AUTO_RENEW` | Optional renewal flag |

### Managed app inputs

| Input | Meaning |
| --- | --- |
| `INPUT_APP_NAME` | App name stored in the dashboard and used as the build prefix for direct flows |
| `INPUT_APP_ID` | Existing dashboard app ID for `app-deploy-direct` and `app-deploy` |
| `INPUT_PUBLIC_DOMAIN` | Public domain to bind in dashboard routing |
| `INPUT_PROXY_TARGET_PORT` | Target port inside the app stack, such as `80` or `8080` |
| `INPUT_USE_TLS` | Whether dashboard-managed TLS should be enabled for `INPUT_PUBLIC_DOMAIN` |
| `INPUT_HEALTHCHECK_URL` | Optional URL checked after app deployment |
| `INPUT_TAG` | Optional explicit image tag instead of `sha-<commit>` |

## Recommended `.paas/config.yml`

```yaml
server: production

defaults:
  INPUT_APP_NAME: paas-dashboard
  INPUT_DASHBOARD_URL: http://127.0.0.1:7000
  INPUT_DASHBOARD_USER: admin
  INPUT_DASHBOARD_PASS: ""
  INPUT_CERTBOT_EMAIL: ""
  INPUT_CERTBOT_STAGING: "false"
  INPUT_CERTBOT_AUTO_RENEW: "true"
  INPUT_REGISTRY_HOST: <REGISTRY_HOST>
  INPUT_IMAGE_REPOSITORY: <REGISTRY_NAMESPACE>/<REPOSITORY>
  # Example app-* overrides. Uncomment and adjust when deploying a managed app.
  # INPUT_APP_ID: "<DASHBOARD_APP_ID>"
  # INPUT_PUBLIC_DOMAIN: "app.example.com"
  # INPUT_PROXY_TARGET_PORT: "8080"
  # INPUT_USE_TLS: "true"
  # INPUT_HEALTHCHECK_URL: "https://app.example.com/health"
  # INPUT_TAG: ""

extensions_dir: .paas/extensions
```

Keep secrets outside git:

```bash
export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
```

## One-time server prerequisites

Prepare the server before running any extension:

```bash
ssh root@<SERVER_HOST>
docker --version
docker compose version
nginx -v
sudo nginx -t
certbot --version
sudo mkdir -p /opt/stacks
ls -ld /opt/stacks
```

The server must provide:

- Docker Engine and `docker compose`
- host `nginx`
- host `certbot`
- writable `/opt/stacks`
- Docker network creation rights for the dashboard controller

## Wrapper usage

Use the project wrapper on Windows / Git Bash and optionally on Linux/macOS:

```bash
bash ./.paas/run.sh bootstrap-direct
```

The wrapper applies this priority:

1. exported `INPUT_*` variables
2. values from `.paas/config.yml`
3. extension defaults

### One-shot `INPUT_*` overrides

You can override any value for a single run without editing `.paas/config.yml`.

Examples:

```bash
INPUT_APP_NAME=my-app \
INPUT_PUBLIC_DOMAIN=app.example.com \
INPUT_PROXY_TARGET_PORT=8080 \
INPUT_USE_TLS=true \
bash ./.paas/run.sh app-bootstrap-direct
```

```bash
INPUT_APP_ID=01HTEXAMPLEAPPID \
INPUT_APP_NAME=my-app \
INPUT_PUBLIC_DOMAIN=app.example.com \
INPUT_PROXY_TARGET_PORT=8080 \
INPUT_USE_TLS=true \
bash ./.paas/run.sh app-deploy-direct
```

```bash
INPUT_APP_ID=01HTEXAMPLEAPPID \
INPUT_APP_NAME=my-app \
INPUT_IMAGE_REPOSITORY=myteam/my-app \
INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>" \
INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>" \
bash ./.paas/run.sh app-deploy
```

This works because `bash ./.paas/run.sh ...` reads exported or inline `INPUT_*` variables first, then falls back to `.paas/config.yml`.

Validate the extensions before the first real run:

```bash
./paas.exe validate bootstrap-direct
./paas.exe validate deploy-direct
./paas.exe validate deploy
./paas.exe validate app-bootstrap-direct
./paas.exe validate app-deploy-direct
./paas.exe validate app-deploy
```

## `bootstrap-direct`

Use this flow on a fresh server or when reinstalling the dashboard runtime from scratch.

```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/<KEY_FILE>

./paas.exe validate bootstrap-direct
bash ./.paas/run.sh bootstrap-direct --dry-run
bash ./.paas/run.sh bootstrap-direct
```

What it does:

1. uploads the tracked repository contents to `/tmp/build-<INPUT_APP_NAME>` on the server
2. builds the dashboard image on the server
3. renders the root `docker-compose.yml` into `/opt/<INPUT_APP_NAME>/docker-compose.yml`
4. starts or replaces the dashboard runtime with `docker compose up -d --remove-orphans`
5. waits until `INPUT_DASHBOARD_URL/api/health` responds
6. optionally seeds platform TLS settings through `PUT /api/settings` if `INPUT_CERTBOT_EMAIL` is not empty

Operational note:

- this flow can initialize platform settings in the runtime database.

After bootstrap, verify on the server:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" \
  "${INPUT_DASHBOARD_URL}/api/health"

docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" ps
docker ps --format '{{.Names}}'
```

## `deploy-direct`

Use this flow for routine updates without a registry hop.

```bash 
./paas.exe validate deploy-direct
bash ./.paas/run.sh deploy-direct --dry-run
bash ./.paas/run.sh deploy-direct
```

What it does:

1. uploads source to the server
2. builds a new local dashboard image tag
3. re-renders `/opt/<INPUT_APP_NAME>/docker-compose.yml`
4. runs `docker compose up -d --remove-orphans`
5. verifies the internal API on `INPUT_DASHBOARD_URL`

Operational note:

- this flow updates the dashboard runtime but does not write platform settings through `/api/settings`.
- use it for routine upgrades when GUI or API-managed platform settings should remain untouched.

## `deploy`

Use this flow when you want the dashboard image pushed to a registry as part of the update.

```bash
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"

./paas.exe validate deploy
bash ./.paas/run.sh deploy --dry-run
bash ./.paas/run.sh deploy
```

What it does:

1. uploads source to the server
2. logs in to the registry on the server
3. builds and tags the image
4. pushes both `sha-<commit>` and `main`
5. re-renders `/opt/<INPUT_APP_NAME>/docker-compose.yml` with the registry image
6. updates the runtime and verifies the internal API

## Managed app flows

These aliases use the dashboard API and are intended for ordinary application repositories, not for the dashboard runtime itself.

### `app-bootstrap-direct`

Use this flow to create a brand new app in the dashboard from the current repository and deploy it immediately.

```bash
export INPUT_APP_NAME=my-app
export INPUT_PUBLIC_DOMAIN=app.example.com
export INPUT_PROXY_TARGET_PORT=8080
export INPUT_USE_TLS=true

./paas.exe validate app-bootstrap-direct
bash ./.paas/run.sh app-bootstrap-direct --dry-run
bash ./.paas/run.sh app-bootstrap-direct
```

What it does:

1. uploads the tracked repository contents to the server
2. builds the app image on the server
3. renders the repo `docker-compose.yml`
4. optionally seeds platform TLS settings when `INPUT_CERTBOT_EMAIL` is set
5. creates a new dashboard app through `POST /api/apps`
6. applies routing through `PUT /api/apps/<id>/config`
7. triggers deployment through `POST /api/apps/<id>/deploy`

### `app-deploy-direct`

Use this flow to update an existing dashboard app by rebuilding from source on the server.

```bash
export INPUT_APP_ID=<DASHBOARD_APP_ID>
export INPUT_APP_NAME=my-app
export INPUT_PUBLIC_DOMAIN=app.example.com
export INPUT_PROXY_TARGET_PORT=8080
export INPUT_USE_TLS=true

./paas.exe validate app-deploy-direct
bash ./.paas/run.sh app-deploy-direct --dry-run
bash ./.paas/run.sh app-deploy-direct
```

What it does:

1. uploads the tracked repository contents to the server
2. builds the app image on the server
3. renders the repo `docker-compose.yml`
4. updates the existing dashboard app through `PUT /api/apps/<id>`
5. reapplies routing through `PUT /api/apps/<id>/config`
6. triggers deployment through `POST /api/apps/<id>/deploy`

### `app-deploy`

Use this flow to update an existing dashboard app while also pushing the image to a registry.

```bash
export INPUT_APP_ID=<DASHBOARD_APP_ID>
export INPUT_APP_NAME=my-app
export INPUT_IMAGE_REPOSITORY=myteam/my-app
export INPUT_PUBLIC_DOMAIN=app.example.com
export INPUT_PROXY_TARGET_PORT=8080
export INPUT_USE_TLS=true
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"

./paas.exe validate app-deploy
bash ./.paas/run.sh app-deploy --dry-run
bash ./.paas/run.sh app-deploy
```

What it does:

1. uploads the tracked repository contents to the server
2. logs in to the registry on the server
3. builds and pushes the image
4. renders the repo `docker-compose.yml` with the registry image reference
5. updates the existing dashboard app through `PUT /api/apps/<id>`
6. reapplies routing through `PUT /api/apps/<id>/config`
7. triggers deployment through `POST /api/apps/<id>/deploy`

## Internal-only operations

The dashboard is not meant to be published through host `nginx`.

- API from the server: `http://127.0.0.1:7000`
- GUI only when `DASHBOARD_MODE=gui`
- recommended GUI access from the local machine:

```bash
ssh -N -L 7500:127.0.0.1:7000 -i ~/.ssh/<KEY_FILE> root@<SERVER_HOST>
```

Then open:

```text
http://127.0.0.1:7500
```

Do not run the tunnel command from inside an existing shell on the server. Run it on your local workstation where the browser is open.

### Temporary switch to GUI mode

The runtime compose currently defaults to `DASHBOARD_MODE: api`, so `bootstrap-direct`, `deploy-direct`, and `deploy` will always bring the controller up in API mode.

To temporarily switch the running dashboard into GUI mode on the server:

```bash
grep -n "DASHBOARD_MODE" /opt/${INPUT_APP_NAME}/docker-compose.yml
sed -i 's/DASHBOARD_MODE: api/DASHBOARD_MODE: gui/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

Verify on the server:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" \
  "${INPUT_DASHBOARD_URL}/api/health"
curl -I http://127.0.0.1:7000/
```

Expected result:

- `/api/health` reports `"mode":"gui"`
- `/` no longer returns `404`

After that, create the SSH tunnel on the local machine:

```bash
ssh -N -L 7500:127.0.0.1:7000 -i ~/.ssh/<KEY_FILE> root@<SERVER_HOST>
```

And open:

```text
http://127.0.0.1:7500
```

To return the controller to hardened API mode:

```bash
sed -i 's/DASHBOARD_MODE: gui/DASHBOARD_MODE: api/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

## Platform TLS settings

The settings page remains in place for managed application domains only.

Seed them during bootstrap if desired:

```yaml
INPUT_CERTBOT_EMAIL: ops@example.com
INPUT_CERTBOT_STAGING: "false"
INPUT_CERTBOT_AUTO_RENEW: "true"
```

These values are written through `/api/settings` and are later used only when managed apps enable proxy TLS.

Important distinction:

- `.paas/config.yml` is a deployment input source,
- `/settings` reads persisted runtime state from the dashboard database.

Recommended model:

1. use `bootstrap-direct` to seed initial platform settings,
2. use `deploy-direct` for routine dashboard upgrades,
3. manage later settings changes through the GUI or API without overwriting them on every deploy.

## Verification commands

Run these on the server:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" \
  "${INPUT_DASHBOARD_URL}/api/health"

docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" ps
docker logs --tail 120 "${INPUT_APP_NAME}-dashboard-1"
```

If you are checking GUI access through the SSH tunnel, run these on the local machine while the tunnel process is active:

```bash
curl -I http://127.0.0.1:7500
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" \
  http://127.0.0.1:7500/api/health
```

If the Compose-generated container name differs, inspect it with:

```bash
docker ps --format '{{.Names}}'
```

## Troubleshooting

### API never becomes ready

Check:

```bash
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" logs --tail 120
ss -ltnp | grep 7000
```

### SSH upload works but remote bash fails

Use the wrapper:

```bash
bash ./.paas/run.sh <extension>
```

It strips problematic Windows environment variables before invoking `paas.exe`.

### TLS for managed apps fails

Verify host prerequisites:

```bash
sudo nginx -t
sudo systemctl reload nginx
certbot certificates
```

### Read-only container validation

The dashboard compose file enables a read-only root filesystem. Runtime writes must go to mounted host paths such as `/opt/stacks`, `/etc/nginx`, and the Let's Encrypt directories. If startup fails, inspect the logs for an unexpected write target.
