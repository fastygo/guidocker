# Backlog & deployment notes

## Improvements for Dockerfile

- **Install Docker CLI and Compose plugin**  
  The app runs `docker compose` inside the container; the current image only has the binary and no Docker client. Add to the final stage (e.g. Docker CE repo + `docker-ce-cli`, `docker-compose-plugin`) so deploy/stop/restart work out of the box.

- **Pin Docker CLI version**  
  If adding Docker repo, pin package versions for reproducible builds.

- **Document required runtime**  
  Add a short comment or README note that the container must be run with `--group-add <docker_gid>` and access to `/var/run/docker.sock` for PaaS features to work.

- **Optional: create docker group in image**  
  For clarity, create a group with a fixed GID (e.g. 999) and add `paas` to it; document that the host should run the container with `--group-add 999` (or match host’s docker GID). This makes “permission denied” on the socket less likely when users copy-paste run commands.

---

## Improvements for Makefile

- **Add `--group-add` to `docker-run`**  
  The container process runs as user `paas`; the Docker socket on the host is owned by `root:docker`. Without `--group-add <gid>`, deploy/stop/restart fail with “permission denied” on the Docker API. Add a variable, e.g. `DOCKER_GID ?= 999`, and pass `--group-add $(DOCKER_GID)` in the `docker run` command. Document that users should set `DOCKER_GID` to their host’s docker GID (`getent group docker`).

- **Document `make docker-run` prerequisites**  
  In `help` or a comment: ensure `/opt/stacks` exists and has correct ownership for the container user (or create it with the right UID/GID). Optionally add a target that creates `/opt/stacks` and sets ownership from the container’s `paas` UID.

- **Optional: target to run with current host docker GID**  
  e.g. `docker-run-auto` that runs `getent group docker` and passes that GID to `--group-add` so `make docker-run-auto` works on most Linux hosts without editing the Makefile.

- **Optional: healthcheck**  
  Add a `docker-health` or use Docker HEALTHCHECK in the image so orchestrators can see when the dashboard is ready.

---

## Problems encountered and solutions (VPS deployment)

### 1. `make` not found

- **Symptom:** `Command 'make' not found` when running `make docker-build`.
- **Solution:** Install make or run Docker directly:
  - `apt update && apt install -y make`
  - Or: `docker build -f Dockerfile -t dashboard:latest .`

### 2. Container in restart loop – BoltDB permission denied

- **Symptom:** Container status `Restarting (1)`; logs: `Failed to initialize BoltDB repository: open /opt/stacks/.paas.db.tmp: permission denied`.
- **Cause:** `/opt/stacks` on the host was created by root; the container runs as user `paas` and could not create files there.
- **Solution:**
  ```bash
  docker stop dashboard
  docker run --rm --entrypoint "" dashboard:latest id -u paas   # get UID (e.g. 100)
  sudo chown -R 100:100 /opt/stacks   # use UID from previous command
  docker start dashboard
  ```

### 3. POST /api/apps 400 Bad Request

- **Symptom:** Creating a new app in the form returned 400.
- **Cause:** Validation requires non-empty app name and compose YAML containing the literal string `services:`.
- **Solution:** Fill “App name” and ensure the compose textarea contains valid YAML with a `services:` key (e.g. minimal compose with one service).

### 4. PUT /api/containers/… and POST …/deploy 500 – Docker CLI missing

- **Symptom:** 500 when clicking Start/Deploy; logs showed `exec: "docker": executable file not found in $PATH`.
- **Cause:** The dashboard image did not include the Docker CLI; the app runs `docker compose` via `exec`.
- **Solution:** Rebuild the image with Docker CLI and Compose plugin installed in the Dockerfile (Docker CE repo + `docker-ce-cli`, `docker-compose-plugin`), then redeploy the dashboard container.

### 5. Permission denied on Docker socket

- **Symptom:** After adding Docker CLI, deploy failed with: `unable to get image '...': permission denied while trying to connect to the docker API at unix:///var/run/docker.sock`.
- **Cause:** The process inside the container runs as `paas`, which is not in the host’s `docker` group, so it cannot use the mounted socket.
- **Solution:** Run the container with the host’s docker group GID:
  ```bash
  getent group docker   # note the GID (e.g. 999)
  docker stop dashboard && docker rm dashboard
  docker run -d ... --group-add 999 -v /var/run/docker.sock:/var/run/docker.sock ... dashboard:latest
  ```
  (Use the same `docker run` options as before, only add `--group-add <gid>`.)

### 6. Compose validation: `args` not allowed

- **Symptom:** Deploy/stop/restart failed with: `validating .../docker-compose.yml: services.web additional properties 'args' not allowed`.
- **Cause:** The sample compose for hashicorp/http-echo used the top-level key `args` under the service; the Compose validator in use does not allow that.
- **Solution:** Use `command` instead of `args` in the service definition, e.g.:
  ```yaml
  services:
    web:
      image: hashicorp/http-echo
      command: ["-listen=:80", "-text=Hello World"]
      ports:
        - "8080:80"
  ```
  Update the app’s compose in the UI (or in `/opt/stacks/<app-id>/docker-compose.yml`) and save, then deploy again.

### 7. Obsolete `version` in compose

- **Symptom:** Log warning: `the attribute 'version' is obsolete, it will be ignored`.
- **Solution:** Remove the `version: "3.9"` (or similar) line from the top of `docker-compose.yml` files; it is no longer required by Docker Compose v2.

### 8. “Failed to load app tailwindcss: app not found”

- **Symptom:** Log message when opening or listing apps.
- **Cause:** A reference (e.g. sidebar or bookmark) points to an app ID `tailwindcss` that does not exist in the database.
- **Solution:** Remove or fix the link that points to that app ID; no change needed in Dockerfile/Makefile.

---

## Commands used during deployment (summary)

```bash
# Clone repo and branch
git clone -b codex/paas-mvp-fac1 https://github.com/fastygo/guidocker.git
cd guidocker/dashboard

# Install make (if needed)
apt update && apt install -y make

# Build and run
make docker-build
make docker-run

# Fix /opt/stacks permissions (after permission denied on .paas.db)
docker stop dashboard
docker run --rm --entrypoint "" dashboard:latest id -u paas
sudo chown -R <UID>:<UID> /opt/stacks
docker start dashboard

# Re-run with Docker socket access (after adding Docker CLI to image)
getent group docker
docker stop dashboard && docker rm dashboard
docker run -d --name dashboard --restart unless-stopped -p 3000:3000 \
  --group-add <DOCKER_GID> \
  -v /var/run/docker.sock:/var/run/docker.sock -v /opt/stacks:/opt/stacks \
  -e SERVER_HOST=0.0.0.0 -e PAAS_PORT=3000 \
  -e PAAS_ADMIN_USER=admin -e PAAS_ADMIN_PASS=admin@123 \
  -e STACKS_DIR=/opt/stacks -e BOLT_DB_FILE=/opt/stacks/.paas.db \
  dashboard:latest

# Inspect logs
docker logs -f dashboard
docker logs --tail 50 dashboard
```
