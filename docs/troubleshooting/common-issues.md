# Common Issues And Debugging

## Quick Checks

Start with these commands on the server:

```bash
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/health"
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" ps
docker ps --format '{{.Names}}'
```

## GUI Returns `404`

Most common reason:

- the runtime is in `DASHBOARD_MODE=api`.

What to check:

```bash
grep -n "DASHBOARD_MODE" /opt/${INPUT_APP_NAME}/docker-compose.yml
curl -u "${INPUT_DASHBOARD_USER}:${INPUT_DASHBOARD_PASS}" "${INPUT_DASHBOARD_URL}/api/health"
curl -I http://127.0.0.1:7000/
```

Expected behavior:

- in `api` mode, `/` returns `404`,
- in `gui` mode, `/` and `/login` should respond normally.

## `/settings` Page Is Empty

Most common reason:

- the runtime database has no saved platform settings.

Typical cause:

- `.paas/config.yml` contains values,
- but the latest deployment flow did not write `PUT /api/settings`.

Remember:

- deployment inputs are not automatically runtime persisted settings,
- `deploy-direct` does not seed platform settings,
- `bootstrap-direct` can seed them when `INPUT_CERTBOT_EMAIL` is provided.

## API Health Works But GUI Does Not

This usually means the controller is healthy, but running in API-only mode.

The health endpoint can succeed while the HTML interface is intentionally disabled.

## Deployment Finished But Expected Settings Did Not Change

Check which flow was used:

- `bootstrap-direct` can seed initial platform settings,
- `deploy-direct` updates the runtime but does not write platform settings,
- GUI changes can therefore survive routine runtime upgrades.

## Managed App TLS Fails

Check:

- domain DNS,
- app public domain,
- proxy target port,
- per-app TLS flag,
- platform certbot settings,
- host nginx validity,
- certbot status and logs.

Useful commands:

```bash
sudo nginx -t
sudo systemctl reload nginx
sudo certbot certificates
docker logs --tail 120 "${INPUT_APP_NAME}-dashboard-1"
```

## Tunnel Works But Browser Behavior Is Strange

Use one local tunnel endpoint consistently.

Recommended:

```bash
ssh -N -L 7500:127.0.0.1:7000 -i ~/.ssh/<KEY_FILE> root@<SERVER_HOST>
```

Then use only:

```text
http://127.0.0.1:7500
```

Avoid mixing different local aliases such as `localhost` in one place and `127.0.0.1` in another during the same workflow.

## Runtime Uses Read-Only Root Filesystem

If startup or runtime actions fail unexpectedly, confirm the process writes only to mounted writable paths.

Important writable locations include:

- `/opt/stacks`
- `/etc/nginx`
- `/etc/letsencrypt`
- `/var/lib/letsencrypt`
- `/var/log/letsencrypt`
- `/tmp`

## FAQ

### Why does a filled `.paas/config.yml` not automatically populate the GUI?

Because `.paas/config.yml` is a deployment input source, while the GUI reads runtime state from the database.

### Why use POST forms instead of fetch calls?

To make operator actions deterministic through redirects, server-side validation, and browser-native form submission semantics.

### Why keep both API and GUI?

Because operators and automation need different interfaces, but both can share the same domain logic and persistence.
