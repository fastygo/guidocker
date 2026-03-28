# Internal Dashboard Deployment Guide

This `.paas` folder now manages the dashboard itself as an internal-only controller.

Target runtime model:

- dashboard API and optional GUI are available only on the server at `http://127.0.0.1:7000`
- public ingress belongs only to managed applications
- dashboard installation and updates happen directly through Docker on the server
- the dashboard never deploys itself through its own API

## Supported flows

| Extension | Purpose |
| --- | --- |
| `bootstrap-direct` | First install on a clean server or a full reinstall |
| `deploy-direct` | Update the dashboard by building the image on the server |
| `deploy` | Update the dashboard through a registry-backed image |

## Required inputs

| Input | Meaning |
| --- | --- |
| `INPUT_APP_NAME` | Docker Compose project / runtime directory name, usually `paas-dashboard` |
| `INPUT_DASHBOARD_URL` | Internal dashboard URL, normally `http://127.0.0.1:7000` |
| `INPUT_DASHBOARD_USER` | Dashboard login user |
| `INPUT_DASHBOARD_PASS` | Dashboard login password |
| `INPUT_CERTBOT_EMAIL` | Optional platform TLS email to seed during `bootstrap-direct` |
| `INPUT_CERTBOT_STAGING` | Optional Let's Encrypt staging flag |
| `INPUT_CERTBOT_AUTO_RENEW` | Optional renewal flag |
| `INPUT_REGISTRY_HOST` | Registry host for the `deploy` flow |
| `INPUT_IMAGE_REPOSITORY` | Registry repository for the `deploy` flow |
| `INPUT_REGISTRY_USERNAME` | Registry username for the `deploy` flow |
| `INPUT_REGISTRY_PASSWORD` | Registry password for the `deploy` flow |

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

Validate the extensions before the first real run:

```bash
./paas.exe validate bootstrap-direct
./paas.exe validate deploy-direct
./paas.exe validate deploy
```

## `bootstrap-direct`

Use this flow on a fresh server or when reinstalling the dashboard runtime from scratch.

```bash
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

## `deploy-direct`

Use this flow for routine updates without a registry hop.

```bash
bash ./.paas/run.sh deploy-direct --dry-run
bash ./.paas/run.sh deploy-direct
```

What it does:

1. uploads source to the server
2. builds a new local dashboard image tag
3. re-renders `/opt/<INPUT_APP_NAME>/docker-compose.yml`
4. runs `docker compose up -d --remove-orphans`
5. verifies the internal API on `INPUT_DASHBOARD_URL`

## `deploy`

Use this flow when you want the dashboard image pushed to a registry as part of the update.

```bash
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"

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

## Internal-only operations

The dashboard is not meant to be published through host `nginx`.

- API from the server: `http://127.0.0.1:7000`
- GUI only when `DASHBOARD_MODE=gui`
- recommended GUI access:

```bash
ssh -L 7500:127.0.0.1:7000 root@<SERVER_HOST>
```

Then open:

```text
http://127.0.0.1:7500
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

## Verification commands

Run these on the server:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" \
  "${INPUT_DASHBOARD_URL}/api/health"

docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" ps
docker logs --tail 120 "${INPUT_APP_NAME}-dashboard-1"
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
