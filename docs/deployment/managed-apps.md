# Managed Application Deployment

## Supported Flows

Managed application repositories use the dashboard API rather than deploying the dashboard runtime itself.

Supported flows:

- `app-bootstrap-direct`
- `app-deploy-direct`
- `app-deploy`

## Flow Roles

### `app-bootstrap-direct`

Use this to create a brand new application in the dashboard and deploy it.

This flow can:

- build the app image on the server,
- create the dashboard app record,
- configure routing,
- deploy the app,
- optionally seed platform TLS settings when `INPUT_CERTBOT_EMAIL` is provided.

### `app-deploy-direct`

Use this to update an existing application by rebuilding on the server.

This flow:

- updates the existing app record,
- reapplies routing,
- deploys the updated runtime.

### `app-deploy`

Use this when the app image should also be pushed to a registry.

## Required Runtime Inputs

Common managed-app inputs include:

- `INPUT_APP_NAME`
- `INPUT_APP_ID` for update flows
- `INPUT_PUBLIC_DOMAIN`
- `INPUT_PROXY_TARGET_PORT`
- `INPUT_USE_TLS`
- `INPUT_HEALTHCHECK_URL`

## Routing Configuration

App routing data is separate from platform settings.

The app-level configuration controls:

- public domain,
- proxy target port,
- whether TLS should be enabled for that app.

These values are written into the app configuration via the dashboard API and stored independently from platform-wide TLS settings.

## Platform Settings Versus App Settings

Do not mix these two concerns:

- platform settings: certbot and global TLS automation policy,
- app settings: domain, port, and per-app TLS intent.

An app can request TLS, but certificate issuance still depends on:

- valid platform TLS settings,
- DNS pointing at the host,
- reachable HTTP challenge path,
- working host `nginx` and `certbot`.

## Recommended Operational Pattern

For predictable operations:

1. Initialize dashboard platform settings once.
2. Use managed app flows for ordinary application deployment.
3. Change app routing per application as needed.
4. Avoid using app deployment flows as a substitute for dashboard runtime administration.
