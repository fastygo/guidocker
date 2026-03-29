# Operator FAQ

## Why is the GUI autonomous now?

Because the dashboard should remain usable even when external UI resources or client-side bootstrap calls are unavailable.

The GUI now relies on:

- server-rendered HTML,
- local static assets,
- standard form submissions,
- shared backend state.

## Why use POST requests instead of fetch-based actions?

Because standard browser `POST` forms are more reliable for operator workflows.

They provide:

- native submission semantics,
- clean redirects,
- simpler validation and error handling,
- less dependence on fragile client-side state.

## Why keep both `api` and `gui` modes?

Because the project supports both human operators and automation.

- `gui` mode is for interactive administration.
- `api` mode is for hardened or automation-first operation.

## Why do I see `404` on `/` after deployment?

The dashboard runtime commonly starts in `DASHBOARD_MODE=api`.

That means:

- `/api/health` can be healthy,
- while `/` still returns `404` because HTML routes are intentionally disabled.

## Why are the fields on `/settings` empty if `.paas/config.yml` is filled?

Because `.paas/config.yml` is a deployment input source, not the runtime settings database.

The `/settings` page reads persisted platform settings from BoltDB.

## Which flow writes platform settings into the database?

`bootstrap-direct` can do it, and it writes `PUT /api/settings` only when `INPUT_CERTBOT_EMAIL` is not empty.

`deploy-direct` does not update platform settings.

For managed applications, the same seed-once idea applies:

- `app-bootstrap-direct` may seed platform settings when `INPUT_CERTBOT_EMAIL` is present,
- `app-deploy-direct` re-applies app routing and deploys the app, but does not sync `/api/settings`.

## Why does the runner now ask for confirmation before deploy?

The wrapper now prints the resolved `INPUT_*` values before execution so operators can confirm what will actually be sent to `paas.exe`.

This helps catch mistakes such as:

- empty `INPUT_PUBLIC_DOMAIN`,
- unexpected `INPUT_USE_TLS=false`,
- missing `INPUT_CERTBOT_EMAIL`,
- missing `INPUT_APP_ID` during update flows.

## Can I skip the confirmation prompt in automation?

Yes.

Use either:

- `bash ./.paas/run.sh --yes <extension> ...`
- `PAAS_ASSUME_YES=true bash ./.paas/run.sh <extension> ...`

This preserves the preflight output while skipping the interactive `Y/N` prompt.

## What happens if I change settings in the GUI and then deploy again?

It depends on the deployment flow.

- If the deploy flow does not call `/api/settings`, GUI changes remain.
- If the deploy flow does call `/api/settings`, those fields can be overwritten by deployment inputs.

## What is the recommended model for this project?

Use the seed-once model:

- `bootstrap-direct`: seed initial settings,
- `deploy-direct`: do not touch settings,
- GUI or API: own later runtime changes.

For managed apps, the practical equivalent is:

- use `app-bootstrap-direct` for first creation,
- let it seed platform settings only when intended,
- use `app-deploy-direct` for routine updates without overwriting platform settings.

## How can I confirm what app config was actually stored?

There are now several layers of visibility:

- runner preflight shows the effective deploy inputs before execution,
- app deploy flows print the requested routing payload,
- app deploy flows fetch and print the stored app config after `PUT /api/apps/<id>/config`,
- the app detail page can log a temporary browser console snapshot of DB-backed rendered values and current DOM field values.

## Can I change the dashboard port or hide the GUI without breaking apps?

Yes, as long as persistent state and host-side runtime artifacts remain intact.

Changing:

- port,
- bind address,
- dashboard mode,
- GUI exposure

does not by itself delete:

- app definitions,
- compose stacks,
- certbot state,
- nginx routes.

## Why is nginx/certbot still important if the GUI is internal?

Because the internal dashboard and the public ingress layer serve different purposes.

- dashboard: control plane,
- host nginx: public routing for managed apps,
- host certbot: certificate lifecycle for managed app domains.

## Is the dashboard supposed to deploy itself through its own API?

No.

The dashboard runtime is deployed through the dashboard runtime extensions, not through its own managed app API.
