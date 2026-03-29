# Smart GUI Control Plane

A small Go-based control plane for managing compose-driven applications with:

- a server-rendered operator dashboard,
- a stable JSON API for automation,
- BoltDB-backed runtime state,
- host-managed `nginx` and `certbot` for managed app ingress.

## What This Project Is

This repository is built around an internal dashboard that manages application lifecycle, routing, and TLS automation without requiring the dashboard itself to be public.

The current production model is:

- the dashboard listens internally on `127.0.0.1:7000`,
- the hardened runtime usually starts in `DASHBOARD_MODE=api`,
- the GUI is enabled only for operator sessions,
- remote GUI access is expected through SSH tunneling,
- managed applications continue to work independently of GUI exposure.

## Why The Current Design Exists

After the refactor, the operator experience is intentionally simpler and more reliable:

- GUI pages are server-rendered,
- major mutations use standard browser `POST` forms,
- local static assets are served by the dashboard itself,
- the API remains available for deployment and automation flows,
- runtime settings and app state remain persisted outside transient dashboard restarts.

This reduces:

- browser-side fetch fragility,
- cross-origin and cookie issues during tunnel-based access,
- coupling between deploy-time inputs and live runtime state.

## Main Components

```text
@GUIDocker/
├── dashboard/      # Go control plane, views, static assets, tests
├── docs/           # architecture, operations, runbooks, FAQ, troubleshooting
├── .paas/          # deployment extensions, wrapper, deployment guides
├── website/        # example managed site stack
├── Dockerfile      # dashboard image build
└── docker-compose.yml
```

## Quick Start

### Run locally

```bash
cd dashboard
go test ./...
go build -o dashboard .
PAAS_PORT=7000 PAAS_ADMIN_USER=admin PAAS_ADMIN_PASS=admin@123 ./dashboard
```

Open:

```text
http://localhost:7000
```

Useful local pages:

- `http://localhost:7000/apps`
- `http://localhost:7000/apps/new`
- `http://localhost:7000/settings`
- `http://localhost:7000/api/health`

### Fast local UI iteration

```bash
cd dashboard
DASHBOARD_AUTH_DISABLED=true air
```

## Production Access Model

Recommended production posture:

- keep the dashboard internal-only,
- publish only managed application domains through host `nginx`,
- use `DASHBOARD_MODE=api` by default,
- switch to `gui` only when an operator needs the HTML interface.

Expected internal endpoint:

```text
http://127.0.0.1:7000
```

To use the GUI remotely:

```bash
ssh -N -L 7500:127.0.0.1:7000 root@<SERVER_HOST>
```

Then open:

```text
http://127.0.0.1:7500
```

If the runtime remains in `DASHBOARD_MODE=api`, the tunnel still works, but only `/api/*` routes are intended to be available.

## Deployment Model

There are two distinct operational concerns:

### 1. Dashboard runtime deployment

Use the dashboard runtime flows:

- `bootstrap-direct`
- `deploy-direct`
- `deploy`

Recommended pattern:

- `bootstrap-direct` seeds initial platform settings,
- `deploy-direct` updates the runtime without overwriting those settings.

### 2. Managed application deployment

Use the managed app flows:

- `app-bootstrap-direct`
- `app-deploy-direct`
- `app-deploy`

These flows use the dashboard API to create, update, configure, and deploy ordinary applications.

## Runtime Settings Versus Deploy Inputs

One of the most important project rules:

- `.paas/config.yml` is a deployment input source,
- `/settings` reads persisted runtime platform settings from BoltDB.

That means:

- a filled `.paas/config.yml` does not automatically populate the GUI,
- `bootstrap-direct` can seed platform settings through `PUT /api/settings`,
- `deploy-direct` should not overwrite operator-managed settings during ordinary runtime upgrades.

## Managed TLS

Platform TLS settings belong to managed application domains, not to the dashboard’s own public exposure.

Platform-wide settings include:

- `certbot_email`
- `certbot_enabled`
- `certbot_staging`
- `certbot_auto_renew`
- `certbot_terms_accepted`

Per-app routing still matters separately:

- public domain,
- proxy target port,
- app TLS flag.

Certificate issuance depends on all of these plus valid host-side `nginx`, DNS, and `certbot` behavior.

## Where To Read Next

Start here depending on your goal:

- [`docs/README.md`](docs/README.md): full documentation index
- [`docs/architecture/overview.md`](docs/architecture/overview.md): architecture and design decisions
- [`docs/runbooks/dashboard-operations.md`](docs/runbooks/dashboard-operations.md): operating the dashboard runtime
- [`docs/runbooks/managed-app-operations.md`](docs/runbooks/managed-app-operations.md): deploying managed applications
- [`docs/runbooks/incident-response.md`](docs/runbooks/incident-response.md): fast incident handling
- [`.paas/README.md`](.paas/README.md): deployment extension usage
- [`dashboard/README.md`](dashboard/README.md): dashboard runtime, routes, configuration, and development details
