# Deployment Guide for this repo (level 101)

This guide is written as a practical copy-paste playbook.

## Before you start: single source of truth for variables

Use only these names in files and commands.  
If a value is sensitive, keep it in environment variables, not in the repo.

### Required / common variables

| Placeholder in examples | Actual name to use | Where to keep |
|---|---|---|
| `<APP_NAME>` | `INPUT_APP_NAME` | `.paas/config.yml` |
| `<APP_ID>` | `INPUT_APP_ID` | `.paas/config.yml` (empty first run) |
| `<BOOTSTRAP_DASHBOARD_HOST>` + `<BOOTSTRAP_DASHBOARD_PORT>` | `INPUT_BOOTSTRAP_DASHBOARD_URL` | `.paas/config.yml` |
| `<DASHBOARD_HOST>` + `<DASHBOARD_PORT>` | `INPUT_DASHBOARD_URL` | `.paas/config.yml` |
| `<DASHBOARD_USER>` | `INPUT_DASHBOARD_USER` | `.paas/config.yml` |
| `<DASHBOARD_PASSWORD>` | `INPUT_DASHBOARD_PASS` | `ENV` (export) for sensitive value |
| `<APP_PUBLIC_DOMAIN>` | `INPUT_PUBLIC_DOMAIN` | `.paas/config.yml` |
| `<PROXY_PORT>` (for dashboard API-only deploy use `7000`) | `INPUT_PROXY_TARGET_PORT` | `.paas/config.yml` |
| `<USE_TLS>` (`"true"` / `"false"`) | `INPUT_USE_TLS` | `.paas/config.yml` |
| `<APP_HEALTHCHECK_URL>` | `INPUT_HEALTHCHECK_URL` | `.paas/config.yml` or `ENV` |
| `<CERTBOT_EMAIL>` | `INPUT_CERTBOT_EMAIL` | `.paas/config.yml` |
| `<CERTBOT_STAGING>` (`"true"` / `"false"`) | `INPUT_CERTBOT_STAGING` | `.paas/config.yml` |
| `<CERTBOT_AUTO_RENEW>` (`"true"` / `"false"`) | `INPUT_CERTBOT_AUTO_RENEW` | `.paas/config.yml` |
| `<REGISTRY_HOST>` | `INPUT_REGISTRY_HOST` | `.paas/config.yml` |
| `<REGISTRY_NAMESPACE>/<REPOSITORY>` | `INPUT_IMAGE_REPOSITORY` | `.paas/config.yml` |
| `<REGISTRY_USERNAME>` | `INPUT_REGISTRY_USERNAME` | `ENV` |
| `<REGISTRY_PASSWORD>` | `INPUT_REGISTRY_PASSWORD` | `ENV` |

### Server connection constants (for `~/.config/paas/servers.yml`)

| Placeholder | Name in file |
|---|---|
| `<SERVER_HOST>` | `servers.production.host` |
| `<PATH_TO_PRIVATE_KEY>` | `servers.production.key` |
| `<DASHBOARD_USER>` | `servers.production.dashboard_user` |
| `<DASHBOARD_PASSWORD>` | `servers.production.dashboard_pass` |

### Rule (to avoid confusion)

- **In all `.paas/extensions/*.yml` and runner input flow, use `INPUT_*` names.**
- Keep secrets (`PASSWORD`, `TOKEN`, `SECRET`, private keys) outside git and inject them via `export` / CI secrets.

Example:

```bash
export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
```

Do not use `APP_PUBLIC_DOMAIN`, `APP_PROXY_TARGET_PORT`, `USE_TLS` directly in `.paas/config.yml`.
Use only `INPUT_PUBLIC_DOMAIN`, `INPUT_PROXY_TARGET_PORT`, `INPUT_USE_TLS`.

You can copy this `.paas` into another app and only replace placeholders.

You can use the same `.paas` folder and `paas.exe` in another app:

1) copy the folder `.paas` to the new project,
2) copy the Docker files (`Dockerfile`, `docker-compose.yml`, `.dockerignore`, `.gitattributes`),
3) edit only app-specific values in `.paas/config.yml` and optionally extension defaults,
4) run the same commands.

Never keep real credentials in this file; use placeholders and environment variables.

---

## Table of flow map

| Scenario | Extension file | Use case |
|---|---|---|
| First deploy (new dashboard app) | `.paas/extensions/bootstrap-direct.yml` | App does not exist in dashboard yet |
| Regular updates | `.paas/extensions/deploy-direct.yml` | App already exists in dashboard |
| Registry flow | `.paas/extensions/deploy.yml` | You need registry-based delivery |

Direct flows build the image on the server and update dashboard `compose_yaml` without a registry hop for runtime.

---

## 0) One-time setup (once per machine)

### Build and initialize the runner

```bash
go build -o ./paas.exe ./cmd/paas
./paas.exe init
```

### Configure server profile: `~/.config/paas/servers.yml`

```yaml
servers:
  production:
    host: <SERVER_HOST>
    port: 22
    user: root
    key: <PATH_TO_PRIVATE_KEY>
    dashboard_user: <DASHBOARD_USER>
    dashboard_pass: <DASHBOARD_PASSWORD>
    host_key_check: tofu
```

### Prepare `.paas/config.yml` (generic)

```yaml
server: production

defaults:
  INPUT_APP_NAME: <APP_NAME>
  INPUT_APP_ID: ""
  INPUT_BOOTSTRAP_DASHBOARD_URL: http://127.0.0.1:3000
  INPUT_DASHBOARD_URL: http://127.0.0.1:7000
  INPUT_DASHBOARD_USER: <DASHBOARD_USER>
  INPUT_DASHBOARD_PASS: ""
  INPUT_HEALTHCHECK_URL: ""
  INPUT_PUBLIC_DOMAIN: <APP_PUBLIC_DOMAIN>
  INPUT_PROXY_TARGET_PORT: "7000"
  INPUT_USE_TLS: "false"
  INPUT_CERTBOT_EMAIL: <CERTBOT_EMAIL>
  INPUT_CERTBOT_STAGING: "false"
  INPUT_CERTBOT_AUTO_RENEW: "true"
  INPUT_REGISTRY_HOST: <REGISTRY_HOST>
  INPUT_IMAGE_REPOSITORY: <IMAGE_REPOSITORY>
  
extensions_dir: .paas/extensions
```

If you export secrets in environment, leave `INPUT_DASHBOARD_PASS` and registry passwords empty in file:

```bash
export INPUT_DASHBOARD_USER="<DASHBOARD_USER>"
export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
```

### Check SSH connectivity before running any deploy

```bash
eval "$(ssh-agent -s)"
ssh-add <PATH_TO_PRIVATE_KEY>
ssh root@<SERVER_HOST>
```

If this command opens shell without password, key-based SSH works.

### Check local helpers

```bash
ssh -V
git --version
docker --version
```

### Validate runner and files

```bash
./paas.exe validate bootstrap-direct
./paas.exe validate deploy-direct
./paas.exe list
./paas.exe servers
```

---

## 1) Windows-safe command wrapper (recommended)

On Git Bash in Windows, remote bash can fail because environment includes invalid variable names like `ProgramFiles(x86)`.

Use the project wrapper for every deploy extension. It applies this priority:

1. exported `INPUT_*` variables from the current shell
2. values from `./.paas/config.yml` under `defaults:`
3. extension-level fallback defaults

Recommended command:

```bash
bash ./.paas/run.sh <extension-name>
```

If you still want the low-level manual form, use this wrapper:

```bash
env -i \
  HOME="$HOME" \
  USERPROFILE="$USERPROFILE" \
  HOMEDRIVE="$HOMEDRIVE" \
  HOMEPATH="$HOMEPATH" \
  PATH="$PATH" \
  TERM="${TERM:-xterm-256color}" \
  LANG="${LANG:-en_US.UTF-8}" \
  SSH_AUTH_SOCK="$SSH_AUTH_SOCK" \
  SSH_AGENT_PID="$SSH_AGENT_PID" \
  ./paas.exe run <extension-name>
```

For the registry extension `deploy`, exported credentials still override config values:

```bash
env -i \
  HOME="$HOME" \
  USERPROFILE="$USERPROFILE" \
  HOMEDRIVE="$HOMEDRIVE" \
  HOMEPATH="$HOMEPATH" \
  PATH="$PATH" \
  TERM="${TERM:-xterm-256color}" \
  LANG="${LANG:-en_US.UTF-8}" \
  SSH_AUTH_SOCK="$SSH_AUTH_SOCK" \
  SSH_AGENT_PID="$SSH_AGENT_PID" \
  INPUT_REGISTRY_USERNAME="$INPUT_REGISTRY_USERNAME" \
  INPUT_REGISTRY_PASSWORD="$INPUT_REGISTRY_PASSWORD" \
  ./paas.exe run deploy
```

Linux/macOS: you can run direct commands, wrapper is optional.

### Wrapper examples

```bash
bash ./.paas/run.sh bootstrap-direct
```

```bash
INPUT_USE_TLS=true bash ./.paas/run.sh deploy-direct
```

---

## 2) First deploy flow (`bootstrap-direct`)

### What it is for
Use when dashboard app is not created yet.

### Variables used by this flow (set before running commands)

- `INPUT_APP_NAME` (from `.paas/config.yml`)
- `INPUT_BOOTSTRAP_DASHBOARD_URL` (from `.paas/config.yml`, old GUI controller on `127.0.0.1:3000`)
- `INPUT_DASHBOARD_URL` (from `.paas/config.yml`)
- `INPUT_DASHBOARD_USER` (from `.paas/config.yml`)
- `INPUT_DASHBOARD_PASS` (environment variable is safer for secrets)
- `INPUT_PUBLIC_DOMAIN` (from `.paas/config.yml`)
- `INPUT_PROXY_TARGET_PORT` (from `.paas/config.yml`)
- `INPUT_USE_TLS` (from `.paas/config.yml`, keep `false` for first bootstrap)
- `INPUT_CERTBOT_EMAIL` (from `.paas/config.yml`)
- `INPUT_CERTBOT_STAGING` (from `.paas/config.yml`)
- `INPUT_CERTBOT_AUTO_RENEW` (from `.paas/config.yml`)
- `STEP_IMAGE_TAG` is generated automatically in the flow.

### Ready-to-run setup snippet

```bash
# Fill in your values (example names only)
export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
export INPUT_APP_NAME="<APP_NAME>"
export INPUT_BOOTSTRAP_DASHBOARD_URL="http://127.0.0.1:3000"
export INPUT_DASHBOARD_URL="http://127.0.0.1:7000"
export INPUT_DASHBOARD_USER="<DASHBOARD_USER>"
export INPUT_PUBLIC_DOMAIN="<APP_PUBLIC_DOMAIN>"
export INPUT_PROXY_TARGET_PORT="7000"
export INPUT_USE_TLS="false"
export INPUT_CERTBOT_EMAIL="<CERTBOT_EMAIL>"
export INPUT_CERTBOT_STAGING="false"
export INPUT_CERTBOT_AUTO_RENEW="true"
export INPUT_HEALTHCHECK_URL=""
```

### Beginner step-by-step

```bash
bash ./.paas/run.sh bootstrap-direct --dry-run
```

```bash
bash ./.paas/run.sh bootstrap-direct
```

### What happens internally

- Reads git SHA and builds tag.
- Uploads tracked sources to `/tmp/build-<APP_NAME>` via `git archive | ssh`.
- Builds image on server.
- Renders `docker-compose.yml` with `APP_IMAGE=<app>:<tag>`.
- Creates app through the old bootstrap controller.
- Writes platform TLS settings through `PUT /api/settings` so later TLS enablement is ready.
- Configures domain/port and keeps the first deploy on plain HTTP.
- Triggers first deploy.
- Verifies the new API-only dashboard at `INPUT_DASHBOARD_URL`.

### Post-step action

Take the printed `APP_ID` and place it into `.paas/config.yml` as `INPUT_APP_ID`.

If you want HTTPS after the first bootstrap:

1. Confirm the domain is already reachable over plain HTTP.
2. Set `INPUT_USE_TLS="true"`.
3. Run `deploy-direct` again, or call `PUT /api/apps/<APP_ID>/config` manually through the API.

---

## 3) Regular update flow (`deploy-direct`)

### What it is for
Use when app is already created and you want fast updates.

### Variables used by this flow (set before running commands)

- `INPUT_APP_NAME` (from `.paas/config.yml`)
- `INPUT_APP_ID` (from `.paas/config.yml`, captured on first deploy)
- `INPUT_DASHBOARD_URL` (from `.paas/config.yml`, steady-state API-only endpoint on `127.0.0.1:7000`)
- `INPUT_DASHBOARD_USER` (from `.paas/config.yml`)
- `INPUT_DASHBOARD_PASS` (environment variable recommended)
- `INPUT_TAG` (optional, can pass in environment as override)
- `INPUT_PUBLIC_DOMAIN` (from `.paas/config.yml`)
- `INPUT_PROXY_TARGET_PORT` (from `.paas/config.yml`)
- `INPUT_USE_TLS` (from `.paas/config.yml`)

### Ready-to-run setup snippet

```bash
export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
export INPUT_APP_NAME="<APP_NAME>"
export INPUT_DASHBOARD_URL="http://127.0.0.1:7000"
export INPUT_DASHBOARD_USER="<DASHBOARD_USER>"
export INPUT_APP_ID="<APP_ID>"
export INPUT_PUBLIC_DOMAIN="<APP_PUBLIC_DOMAIN>"
export INPUT_PROXY_TARGET_PORT="7000"
export INPUT_USE_TLS="true"     # set true only after HTTP publication works
export INPUT_TAG="sha-<SHORT_SHA>"   # optional override
```

### Step-by-step

```bash
bash ./.paas/run.sh deploy-direct --dry-run
```

```bash
bash ./.paas/run.sh deploy-direct
```

### What happens internally

- Uploads source and builds image on server.
- Renders compose with local image tag.
- Updates existing dashboard app (`PUT /api/apps/<APP_ID>`).
- Synchronizes public domain, proxy port, and TLS flag via `PUT /api/apps/<APP_ID>/config`.
- Triggers deploy.

---

## 4) Registry flow (`deploy`)

### What it is for
Use when you want artifacts in private/public registry and immutable remote tags.

### Variables used by this flow (set before running commands)

- `INPUT_APP_NAME` (from `.paas/config.yml`)
- `INPUT_DASHBOARD_URL` (from `.paas/config.yml`, steady-state API-only endpoint on `127.0.0.1:7000`)
- `INPUT_DASHBOARD_USER` (from `.paas/config.yml`)
- `INPUT_DASHBOARD_PASS` (env recommended)
- `INPUT_APP_ID` (from `.paas/config.yml`)
- `INPUT_REGISTRY_HOST` (from `.paas/config.yml`)
- `INPUT_IMAGE_REPOSITORY` (from `.paas/config.yml`)
- `INPUT_REGISTRY_USERNAME` (environment)
- `INPUT_REGISTRY_PASSWORD` (environment)
- `INPUT_PUBLIC_DOMAIN` (from `.paas/config.yml`)
- `INPUT_PROXY_TARGET_PORT` (from `.paas/config.yml`)
- `INPUT_USE_TLS` (from `.paas/config.yml`)

### Ready-to-run setup snippet

```bash
export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
export INPUT_APP_NAME="<APP_NAME>"
export INPUT_DASHBOARD_URL="http://127.0.0.1:7000"
export INPUT_DASHBOARD_USER="<DASHBOARD_USER>"
export INPUT_APP_ID="<APP_ID>"
export INPUT_PUBLIC_DOMAIN="<APP_PUBLIC_DOMAIN>"
export INPUT_PROXY_TARGET_PORT="7000"
export INPUT_USE_TLS="true"     # set true only after HTTP publication works
export INPUT_REGISTRY_HOST="<REGISTRY_HOST>"
export INPUT_IMAGE_REPOSITORY="<REGISTRY_NAMESPACE>/<REPOSITORY>"
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
```

### What happens

- GitHub Actions (or runner override) builds image from `Dockerfile`.
- Pushes `sha-<commit>` and `main` tags.
- Renders compose with registry image.
- Updates dashboard app and deploys.

This flow is valid but slower due the registry hop.

---

## 5) Reusable for a new application

To start another app quickly:

1. Copy `.paas/`, `Dockerfile`, `docker-compose.yml`, `paas.exe`.
2. Update only:
   - `INPUT_APP_NAME`
   - `INPUT_BOOTSTRAP_DASHBOARD_URL`
   - `INPUT_DASHBOARD_URL`
   - `INPUT_DASHBOARD_USER`
   - `INPUT_DASHBOARD_PASS`
   - `INPUT_PUBLIC_DOMAIN`
   - `INPUT_CERTBOT_EMAIL`
   - `INPUT_REGISTRY_*` (if needed)
3. Run `bootstrap-direct` once.
4. Copy returned `APP_ID` into new `.paas/config.yml`.
5. For every release run `deploy-direct`.

---

## 6) Verification commands (critical)

### Runner checks

```bash
./paas.exe list
./paas.exe servers
./paas.exe validate bootstrap-direct
./paas.exe validate deploy-direct
./paas.exe validate deploy
```

### Dashboard checks

These commands assume you run them on the server itself or through an SSH tunnel, because `INPUT_DASHBOARD_URL` points to the server-local API-only endpoint `127.0.0.1:7000`.

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" \
  "${INPUT_DASHBOARD_URL}/api/apps/${INPUT_APP_ID}"

curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" \
  "${INPUT_DASHBOARD_URL}/api/apps/${INPUT_APP_ID}/logs?lines=120"
```

### App availability checks

```bash
curl -I "https://<APP_PUBLIC_DOMAIN>/"
curl -I "http://<APP_PUBLIC_DOMAIN>/"
curl "https://<APP_PUBLIC_DOMAIN>/api/health"
```

### Docker visibility

```bash
docker ps
docker ps -a
docker compose -f docker-compose.yml ps
```

```bash
docker logs --tail 120 dashboard
docker logs -f --tail 120 dashboard

docker compose -f docker-compose.yml logs --tail 120
docker compose -f docker-compose.yml logs -f web
```

### Enter and inspect containers

```bash
docker exec -it dashboard /bin/sh
docker exec dashboard sh -lc "ps -ef | head"
docker exec dashboard env | grep DASHBOARD
docker inspect dashboard
docker inspect <CONTAINER_ID_OR_NAME>
```

### Useful health checks

```bash
docker image ls | grep <APP_NAME>
docker network ls
docker volume ls
docker system df
```

---

## 7) Frequent errors and fixes

### A) Missing input / wrong names

Error sample:
`missing required input "public_domain"`

Fix:
- Use `INPUT_PUBLIC_DOMAIN`, not `APP_PUBLIC_DOMAIN`.
- Use `INPUT_PROXY_TARGET_PORT`, not `APP_PROXY_TARGET_PORT`.
- Use `INPUT_USE_TLS`, not `USE_TLS`.

### B) `syntax error near unexpected token '('`

Cause: Windows environment names like `ProgramFiles(x86)` get exported in remote bash.
Fix: always use `env -i ...` wrapper above.

### C) `ssh: Permission denied (publickey)`

Fix: add public key to server and test:

```bash
ssh root@<SERVER_HOST>
```

### D) Container exits immediately

Check:

```bash
docker ps -a --filter "name=web"
docker logs --tail 200 <CONTAINER_NAME_OR_ID>
```

### E) `address already in use`

When deploy fails with `bind: address already in use`:

```bash
ss -ltnp | grep ':80\|:443'
docker ps --filter "publish=80" --filter "publish=443"
```

Fix: stop or remove the old container using the same port.

### F) App does not answer health checks

Check generated compose and local proxy chain:

```bash
docker compose -f docker-compose.yml config
curl -I http://127.0.0.1:80/
```

If the endpoint responds only over HTTPS, enable TLS and check 443 port mapping.

### G) `exec /usr/local/bin/docker-entrypoint: no such file or directory`

Most often this is `CRLF`/line-ending issue in script files.

Inside Git repo:

```bash
cat .gitattributes
```

Ensure shell scripts use LF and avoid Windows line endings for runtime entrypoints.

---

## 8) Quick first-run checklist (copy-paste for new projects)

Use this as a 1-minute runbook before first deploy of any new app.

- [ ] Copy `.paas/`, `Dockerfile`, `docker-compose.yml`, `.dockerignore`, `.gitattributes`, and `paas.exe` into the new project.
- [ ] Fill `.paas/config.yml`: `INPUT_APP_NAME`, `INPUT_BOOTSTRAP_DASHBOARD_URL`, `INPUT_DASHBOARD_URL`, `INPUT_DASHBOARD_USER`, `INPUT_DASHBOARD_PASS`, `INPUT_PUBLIC_DOMAIN`, `INPUT_CERTBOT_EMAIL`, and `INPUT_REGISTRY_*` if using registry flow.
- [ ] Create/verify `~/.config/paas/servers.yml` with correct `server`, `user`, `key`, `dashboard_user`, `dashboard_pass`.
- [ ] Verify SSH key: `ssh root@<SERVER_HOST>`.
- [ ] Validate extensions: `./paas.exe validate bootstrap-direct` and `./paas.exe validate deploy-direct`.
- [ ] Run first deploy: `bash ./.paas/run.sh bootstrap-direct`.
- [ ] Save printed `APP_ID` into `.paas/config.yml` as `INPUT_APP_ID`.
- [ ] Run checks: `curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/apps/${INPUT_APP_ID}"`, `curl -I "http://<APP_PUBLIC_DOMAIN>/"`, `docker logs --tail 120 dashboard`.
- [ ] After plain HTTP works, set `INPUT_USE_TLS=true` and run `deploy-direct` to request the certificate.

From next release onward:

- [ ] Update code.
- [ ] Run update deploy: `bash ./.paas/run.sh deploy-direct`.

---

## 9) Copy-ready template: `.paas/config.yml`

Create this file in every new app and replace values in `<>`:

```yaml
server: production

defaults:
  # App
  INPUT_APP_NAME: <APP_NAME>

  # Old GUI controller used only for first bootstrap
  INPUT_BOOTSTRAP_DASHBOARD_URL: http://127.0.0.1:3000

  # Steady-state API-only dashboard endpoint
  INPUT_DASHBOARD_URL: http://127.0.0.1:7000
  INPUT_DASHBOARD_USER: <DASHBOARD_USER>
  INPUT_DASHBOARD_PASS: <DASHBOARD_PASSWORD>

  # Optional public smoke test. Leave empty for first bootstrap.
  INPUT_HEALTHCHECK_URL: ""

  # App routing for API-only dashboard on port 7000
  INPUT_PUBLIC_DOMAIN: <APP_PUBLIC_DOMAIN>
  INPUT_PROXY_TARGET_PORT: "7000"
  INPUT_USE_TLS: "false"
  INPUT_CERTBOT_EMAIL: <CERTBOT_EMAIL>
  INPUT_CERTBOT_STAGING: "false"
  INPUT_CERTBOT_AUTO_RENEW: "true"

  # Existing dashboard app id for deploy-direct
  # Leave empty for first deploy, then paste app id from runner output
  INPUT_APP_ID: ""

  # Registry settings for deploy (optional)
  INPUT_REGISTRY_HOST: <REGISTRY_HOST>
  INPUT_IMAGE_REPOSITORY: <REGISTRY_NAMESPACE>/<REPOSITORY>

extensions_dir: .paas/extensions
```

Use environment variables for passwords in shell sessions:

```bash
export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
```

### Quick note on local vs server deploy file

- Use production path as default in `.paas` (`docker-compose.yml`).
- If you later add `docker-compose.local.yml`, keep local overrides separate for development only.

### Line endings for shell scripts

To avoid `/usr/bin/env: 'bash\\r': No such file or directory`, keep shell scripts in LF format.
This repo enforces that through `.gitattributes` with:

```gitattributes
*.sh text eol=lf
```
