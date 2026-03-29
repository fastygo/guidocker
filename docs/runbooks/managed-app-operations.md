# Managed App Operations Runbook

## Purpose

Use this runbook for deploying and updating ordinary applications managed by the dashboard.

## Create A New Managed App

Use `app-bootstrap-direct` for a brand new app:

```bash
export INPUT_APP_NAME=my-app
export INPUT_PUBLIC_DOMAIN=app.example.com
export INPUT_PROXY_TARGET_PORT=8080
export INPUT_USE_TLS=true

./paas.exe validate app-bootstrap-direct
bash ./.paas/run.sh app-bootstrap-direct --dry-run
bash ./.paas/run.sh app-bootstrap-direct
```

The runner preflight now shows the effective `INPUT_*` values before execution and asks for confirmation by default.

Expected outcome:

- a new dashboard app record is created,
- routing is configured,
- deployment is triggered.

Operational notes:

- bootstrap may seed platform settings only when `INPUT_CERTBOT_EMAIL` is resolved
- the flow prints whether that seed is enabled or skipped
- the flow prints the requested app routing payload before `PUT /api/apps/<id>/config`
- the flow fetches and prints the stored app config after routing is applied
- `use_tls=true` expresses app HTTPS intent, but platform TLS settings must still exist or be seeded during bootstrap

## Update An Existing Managed App

Use `app-deploy-direct`:

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

The runner preflight makes missing `INPUT_APP_ID`, empty domains, and unexpected TLS intent visible before the deploy starts.

Expected outcome:

- the existing app record is updated,
- routing is re-applied,
- deployment is triggered.

Operational notes:

- `INPUT_APP_ID` is a hard prerequisite
- this flow re-applies app routing but does not sync platform settings through `/api/settings`
- the flow prints the requested routing payload before the config update
- the flow fetches and prints the stored app config after routing is updated

## Non-Interactive Execution

Use these when automation should skip the confirmation prompt:

```bash
bash ./.paas/run.sh --yes app-bootstrap-direct --dry-run
PAAS_ASSUME_YES=true bash ./.paas/run.sh app-deploy-direct --dry-run
```

## Registry-Backed Update

Use `app-deploy` when the image must also be pushed to a registry:

```bash
export INPUT_APP_ID=<DASHBOARD_APP_ID>
export INPUT_APP_NAME=my-app
export INPUT_IMAGE_REPOSITORY=myteam/my-app
export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"

./paas.exe validate app-deploy
bash ./.paas/run.sh app-deploy --dry-run
bash ./.paas/run.sh app-deploy
```

## Routing Checklist

Before enabling TLS for an app, confirm:

- `INPUT_PUBLIC_DOMAIN` is correct,
- `INPUT_PROXY_TARGET_PORT` matches the service inside the compose stack,
- the domain resolves to the host,
- plain HTTP works first.

## Platform TLS Dependency

Per-app TLS depends on platform settings being present.

That means:

- app-level `use_tls=true` is not enough by itself,
- dashboard platform settings and host certbot prerequisites must already be valid.

## If App TLS Does Not Work

Check:

```bash
sudo nginx -t
sudo systemctl reload nginx
sudo certbot certificates
curl -I http://<PUBLIC_DOMAIN>/
curl -I https://<PUBLIC_DOMAIN>/
```

Also check the app record in the dashboard:

- public domain,
- proxy target port,
- TLS flag,
- deployment status.

## Healthcheck Guidance

If `INPUT_HEALTHCHECK_URL` is used, keep it aligned with the app’s public path after deployment.

This is most useful for:

- immediate smoke validation,
- detecting broken deploys early,
- confirming that routing and TLS are both operational.
