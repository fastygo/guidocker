# Ubuntu Real-Server Test Runbook (Production-focused)

This document is a strict order of operations for a clean Ubuntu server, from fresh install to full verification of API/UI flows and Makefile checks.

Scope:
- Validate production runtime behavior (Docker container, API, and Delete flow)
- Validate business logic tests on the server
- Keep local developer-only tooling out of the runbook

## 1) Prepare a clean Ubuntu server

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg git

# Install Docker Engine + Compose plugin
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker

# Make current user able to use docker
sudo usermod -aG docker "$USER"
newgrp docker
```

## 2) Clean workspace and clone repository

```bash
mkdir -p /opt
cd /opt
git clone <REPO_URL> guidocker
cd guidocker
```

If you need a fixed commit:

```bash
git checkout <COMMIT_OR_BRANCH>
```

## 3) Build dashboard image (from clean state)

Run from `dashboard` directory.

```bash
cd dashboard
docker build -f Dockerfile -t paas-dashboard:latest .
```

If you want to use the repo Makefile:

```bash
make docker-build IMAGE_NAME=paas-dashboard IMAGE_TAG=latest
```

## 4) Start container (production startup path)

Auto-detect host Docker group ID and start container:

```bash
make docker-run-auto \
  IMAGE_NAME=paas-dashboard \
  IMAGE_TAG=latest \
  CONTAINER_NAME=paas-dashboard \
  PAAS_ADMIN_USER=admin \
  PAAS_ADMIN_PASS='admin@123' \
  STACKS_DIR=/opt/stacks
```

Equivalent one-shot command (no Makefile needed):

```bash
HOST_DOCKER_GID="$(getent group docker | cut -d: -f3)"
docker run -d \
  --name paas-dashboard \
  --restart unless-stopped \
  --group-add "${HOST_DOCKER_GID:-999}" \
  -p 3000:3000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /opt/stacks:/opt/stacks \
  -e SERVER_HOST=0.0.0.0 \
  -e PAAS_PORT=3000 \
  -e PAAS_ADMIN_USER=admin \
  -e PAAS_ADMIN_PASS=admin@123 \
  -e STACKS_DIR=/opt/stacks \
  -e BOLT_DB_FILE=/opt/stacks/.paas.db \
  paas-dashboard:latest
```

## 5) Smoke verification: service and auth

```bash
docker ps --filter name=paas-dashboard
curl -f http://127.0.0.1:3000/api/apps
```

With basic auth:

```bash
curl -u admin:admin@123 http://127.0.0.1:3000/api/apps
```

## 6) API-level business logic check (create, deploy, delete)

Create app:

```bash
CREATE_PAYLOAD='{"name":"test-app","compose_yaml":"services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"8080:80\""}'
CREATE_RESP=$(curl -s -u admin:admin@123 -H "Content-Type: application/json" \
  -d "$CREATE_PAYLOAD" \
  http://127.0.0.1:3000/api/apps)
APP_ID=$(printf '%s' "$CREATE_RESP" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
echo "APP_ID=$APP_ID"
```

Deploy app:

```bash
curl -s -u admin:admin@123 -X POST \
  "http://127.0.0.1:3000/api/apps/${APP_ID}/deploy"
```

Delete app (hard delete):

```bash
curl -s -u admin:admin@123 -X DELETE \
  "http://127.0.0.1:3000/api/apps/${APP_ID}"
```

Ensure it is fully removed:

```bash
curl -s -o /tmp/delete-check -w '%{http_code}' \
  -u admin:admin@123 \
  "http://127.0.0.1:3000/api/apps/${APP_ID}" ; echo
```

Expected: non-200 (typically 404).

Also verify the stack path is not left behind:

```bash
docker exec -it paas-dashboard sh -lc "ls -la /opt/stacks | sed -n '1,5p'; ls -la /opt/stacks"
```

## 7) Validate UI path for delete flow

```bash
curl -s http://127.0.0.1:3000/apps | head -n 40
curl -s http://127.0.0.1:3000/apps/new | head -n 40
curl -s -u admin:admin@123 http://127.0.0.1:3000/api/apps | head -n 40
```

## 8) Business-logic tests on the real server (no local Go install needed)

Run tests inside official Go container:

```bash
docker run --rm \
  -v "$PWD:/workspace" \
  -w /workspace/dashboard \
  golang:1.22.2-bookworm \
  go test ./...
```

If tests are executed on a host with Go and Makefile:

```bash
make test
```

## 9) Check Makefile production usage

Only use these targets in server flow:

- `docker-build`
- `docker-run`
- `docker-run-auto`
- `docker-stop`
- `docker-logs`
- `docker-shell`

Verify expected production targets are present:

```bash
make help
make -n docker-run
make -n docker-run-auto
```

`make -n docker-run` must include `--group-add $(DOCKER_GID)` and `/var/run/docker.sock` mount.

## 10) Health and maintenance cleanup

```bash
docker logs --tail 100 paas-dashboard
make docker-logs
make docker-stop
docker rm -f paas-dashboard
```

---

Notes:
- Do not install `make` on Windows workstation for this workflow if memory is constrained.
- Run this sequence from clean Ubuntu server instances with Docker available.
- Makefile can keep local-development helpers, but for production validation this runbook uses only Docker runtime targets.
