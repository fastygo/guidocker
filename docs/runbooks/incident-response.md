# Incident Response Runbook

## Purpose

Use this runbook during production incidents affecting the dashboard runtime, operator GUI, deployment flows, or managed app reachability.

## Incident 1: GUI Returns `404`

### Symptoms

- `http://127.0.0.1:7000/` returns `404`
- API endpoints still work
- SSH tunnel is healthy, but the HTML UI is unavailable

### Most Likely Cause

- the runtime is in `DASHBOARD_MODE=api`

### Checks

```bash
grep -n "DASHBOARD_MODE" /opt/${INPUT_APP_NAME}/docker-compose.yml
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/health"
curl -I http://127.0.0.1:7000/
```

### Recovery

```bash
sed -i 's/DASHBOARD_MODE: api/DASHBOARD_MODE: gui/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

## Incident 2: `/settings` Is Empty

### Symptoms

- the settings page renders empty fields
- `.paas/config.yml` contains certbot-related inputs

### Most Likely Cause

- deployment inputs exist, but runtime platform settings were never written into BoltDB

### Checks

1. Confirm whether `bootstrap-direct` was used.
2. Confirm whether `INPUT_CERTBOT_EMAIL` was non-empty at bootstrap time.
3. Confirm whether settings were later saved through `/settings` or `/api/settings`.

### Recovery

- save the settings through the GUI, or
- call `PUT /api/settings`, or
- re-bootstrap intentionally if you want deploy-time initialization again

Operational note:

- `deploy-direct` should not be expected to populate `/settings`.

## Incident 3: Dashboard Deploy Succeeded But GUI Is Still Unavailable

### Symptoms

- `deploy-direct` completed
- health check passed
- browser still cannot open GUI

### Interpretation

The deployment verified the API, not the HTML UI.

### Checks

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/health"
curl -I http://127.0.0.1:7000/
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" ps
```

### Recovery

- verify `DASHBOARD_MODE`,
- switch to `gui` mode if operator access is required,
- reconnect through an SSH tunnel from the local workstation.

## Incident 4: Certbot Renew Fails

### Symptoms

- certificate renewal action fails
- app remains HTTP-only or cert issuance does not complete

### Checks

```bash
sudo nginx -t
sudo systemctl reload nginx
sudo certbot certificates
docker logs --tail 120 "${INPUT_APP_NAME}-dashboard-1"
```

Verify all prerequisites:

- platform certbot settings exist,
- app public domain is saved,
- target port is correct,
- DNS points to the host,
- HTTP challenge path is reachable.

### Recovery

- fix platform settings if missing,
- fix app routing if incorrect,
- restore plain HTTP reachability first,
- retry renewal after nginx validation succeeds.

## Incident 5: App Deploy Succeeded But Domain Is Not Reachable

### Symptoms

- deployment action reports success
- the public domain does not respond correctly

### Checks

```bash
curl -I http://<PUBLIC_DOMAIN>/
curl -I https://<PUBLIC_DOMAIN>/
sudo nginx -t
docker ps --format '{{.Names}}'
```

Inspect:

- app record public domain,
- app record proxy target port,
- app TLS flag,
- app runtime container status.

### Recovery

- correct routing config,
- redeploy the app,
- verify host `nginx` routing,
- verify TLS only after plain HTTP succeeds.

## Incident 6: Tunnel Is Active But Browser Behavior Is Inconsistent

### Symptoms

- login or page behavior differs between sessions
- some actions appear unreliable

### Most Likely Cause

- mixed use of `localhost` and `127.0.0.1`
- multiple tunnel endpoints used interchangeably

### Recovery

Use one consistent local endpoint:

```bash
ssh -N -L 7500:127.0.0.1:7000 -i ~/.ssh/<KEY_FILE> root@<SERVER_HOST>
```

Then use only:

```text
http://127.0.0.1:7500
```

## Escalation Checklist

Before escalating, collect:

- output of `/api/health`
- `docker compose ps`
- dashboard container logs
- `nginx -t`
- `certbot certificates`
- the active `DASHBOARD_MODE`
- the exact deploy flow that was used
