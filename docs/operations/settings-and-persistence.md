# Settings And Persistence

## Two Different Configuration Layers

The project has two distinct configuration layers:

1. deployment-time inputs,
2. runtime persisted settings.

Confusing these two layers is the most common source of misunderstanding.

## Deployment-Time Inputs

Files such as `.paas/config.yml` are inputs to deployment flows.

They are used by:

- `bash ./.paas/run.sh ...`
- the selected extension under `.paas/extensions`
- `paas.exe run ...`

These values influence what the deployment flow sends to the server and API during that execution.

They are not automatically the same thing as what the running dashboard later shows in its GUI.

## Runtime Persisted Settings

The `/settings` page reads platform settings from the dashboard persistence layer, not directly from `.paas/config.yml`.

Those settings are stored in BoltDB and loaded at runtime by the platform settings service.

Examples:

- certbot email,
- certbot enabled flag,
- staging flag,
- auto-renew flag,
- terms accepted flag.

## Why `/settings` Can Be Empty While `.paas/config.yml` Is Filled

This happens when deployment inputs were present, but no deployment flow wrote them into runtime storage.

Typical case:

- `.paas/config.yml` contains certbot-related inputs,
- `deploy-direct` is used,
- `deploy-direct` does not call `PUT /api/settings`,
- therefore the database still has empty platform settings,
- so the `/settings` page renders empty fields.

## Seed-Once Model

The recommended model for this project is:

- `bootstrap-direct`: seed initial platform settings,
- `deploy-direct`: do not overwrite settings,
- GUI or API: own runtime updates afterwards.

This gives a clean separation:

- bootstrap initializes,
- routine deploys upgrade,
- operators manage live settings without routine deploys clobbering them.

## What Happens On The Next Deploy

### If the deploy flow does not write `/api/settings`

Current database values stay as they are.

That means:

- a checkbox turned off in the GUI stays off,
- a changed email stays changed,
- routine runtime upgrades do not revert those values.

### If the deploy flow does write `/api/settings`

The database is updated from the payload sent by that deployment.

That means:

- a checkbox enabled in deployment inputs can be re-enabled,
- a checkbox disabled in deployment inputs can be turned off again,
- manual GUI changes may be overwritten by the next deploy.

## Source Of Truth Strategies

Choose one strategy and document it for operators:

### Strategy A: Deploy Is Source Of Truth

Every deployment writes platform settings explicitly.

Use this if:

- all runtime policy is centrally controlled in deployment config,
- manual GUI changes should not survive the next deploy.

### Strategy B: GUI Is Source Of Truth

Routine deploys never touch platform settings.

Use this if:

- operators adjust live settings manually,
- deployment should focus only on runtime updates.

### Strategy C: Seed Once, Then Manual Control

Bootstrap seeds the initial settings, and later deploys leave them alone.

This is the recommended strategy for the current project because it matches the separation between `bootstrap-direct` and `deploy-direct`.

## App State Persistence

Application data remains independent from whether the GUI is exposed.

Persistent state includes:

- BoltDB records,
- compose definitions,
- runtime directories under `/opt/stacks`,
- host nginx and certbot artifacts.

Because of that, applications continue to work independently as long as those persisted artifacts remain intact.
