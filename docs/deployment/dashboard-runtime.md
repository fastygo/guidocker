# Dashboard Runtime Deployment

## Supported Flows

The dashboard runtime has three supported deployment flows:

- `bootstrap-direct`
- `deploy-direct`
- `deploy`

## Flow Intent

### `bootstrap-direct`

Use this for first install or full re-bootstrap of the dashboard runtime.

What it does:

1. Uploads tracked repository content to the server.
2. Builds the dashboard image on the server.
3. Renders runtime compose into `/opt/<INPUT_APP_NAME>/docker-compose.yml`.
4. Starts the dashboard runtime.
5. Verifies `INPUT_DASHBOARD_URL/api/health`.
6. Optionally seeds platform TLS settings through `PUT /api/settings`.

Important rule:

- `bootstrap-direct` can initialize platform settings in the database, but only when `INPUT_CERTBOT_EMAIL` is not empty.

### `deploy-direct`

Use this for routine dashboard updates without a registry hop.

What it does:

1. Uploads source to the server.
2. Builds a fresh local image tag on the server.
3. Re-renders the runtime compose file.
4. Restarts with `docker compose up -d --remove-orphans`.
5. Verifies the internal API.

Important rule:

- `deploy-direct` updates the dashboard runtime, but does not write platform settings into the database.

### `deploy`

Use this when the dashboard image should also be pushed to a registry during deployment.

It behaves similarly to `deploy-direct`, but adds registry login, push, and registry-backed compose rendering.

## Settings Seeding Behavior

The distinction between `bootstrap-direct` and `deploy-direct` is important:

- `bootstrap-direct`: may seed platform settings,
- `deploy-direct`: should not touch platform settings,
- GUI changes to settings therefore survive routine runtime deployments.

This makes the common operational model predictable:

1. Bootstrap once with initial settings.
2. Change settings later through the GUI or API.
3. Use `deploy-direct` for runtime upgrades without overwriting those settings.

## Why This Split Is Useful

Without this split, every dashboard redeploy could unintentionally overwrite operator-managed TLS settings.

The current design allows:

- initial provisioning from deployment inputs,
- later operator ownership of runtime settings,
- routine upgrades without configuration churn.

## Dry Run

Use `--dry-run` before a real deployment to validate the extension execution plan and required inputs without applying the real steps.

Typical examples:

```bash
./paas.exe validate bootstrap-direct
bash ./.paas/run.sh bootstrap-direct --dry-run
```

```bash
./paas.exe validate deploy-direct
bash ./.paas/run.sh deploy-direct --dry-run
```

## GUI Mode After Deployment

The rendered runtime compose defaults to API-only mode for hardened operation.

That means:

- API health can be successful,
- while `/` still returns `404` because GUI pages are intentionally disabled.

To temporarily enable GUI mode on the server:

```bash
sed -i 's/DASHBOARD_MODE: api/DASHBOARD_MODE: gui/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

Then verify:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/health"
curl -I http://127.0.0.1:7000/
```
