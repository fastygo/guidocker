# Dashboard Operations Runbook

## Purpose

Use this runbook for day-to-day operation of the dashboard runtime itself.

## First Bootstrap

Use this when installing the dashboard on a clean server or when intentionally reinitializing the runtime.

```bash
./paas.exe validate bootstrap-direct
bash ./.paas/run.sh bootstrap-direct --dry-run
bash ./.paas/run.sh bootstrap-direct
```

Expected outcome:

- runtime compose is rendered under `/opt/${INPUT_APP_NAME}`,
- dashboard container starts,
- API health becomes available,
- platform settings are seeded if `INPUT_CERTBOT_EMAIL` is not empty.

## Routine Dashboard Update

Use this for normal runtime upgrades without overwriting platform settings.

```bash
./paas.exe validate deploy-direct
bash ./.paas/run.sh deploy-direct --dry-run
bash ./.paas/run.sh deploy-direct
```

Expected outcome:

- dashboard image is rebuilt,
- compose is refreshed,
- runtime is restarted,
- API health is checked,
- existing platform settings remain unchanged.

## Enable GUI Temporarily

If the runtime is healthy but `/` returns `404`, switch from API-only mode to GUI mode:

```bash
grep -n "DASHBOARD_MODE" /opt/${INPUT_APP_NAME}/docker-compose.yml
sed -i 's/DASHBOARD_MODE: api/DASHBOARD_MODE: gui/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

Verify:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/health"
curl -I http://127.0.0.1:7000/
```

## Return To API Mode

```bash
sed -i 's/DASHBOARD_MODE: gui/DASHBOARD_MODE: api/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

## Access The GUI Through A Tunnel

Run on the local workstation:

```bash
ssh -N -L 7500:127.0.0.1:7000 -i ~/.ssh/<KEY_FILE> root@<SERVER_HOST>
```

Then open:

```text
http://127.0.0.1:7500
```

## Verify Runtime Health

Run on the server:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/health"
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" ps
docker logs --tail 120 "${INPUT_APP_NAME}-dashboard-1"
```

## If Platform Settings Are Missing

Symptoms:

- `/settings` is empty,
- certbot-related runtime policy appears unset.

Interpretation:

- deployment inputs existed,
- but no flow wrote `PUT /api/settings`,
- or the database did not preserve those values.

Recommended response:

1. Confirm whether `bootstrap-direct` was used with `INPUT_CERTBOT_EMAIL`.
2. If not, save the settings through the GUI or API.
3. Keep using `deploy-direct` for normal updates so those values are not overwritten.
