# Smart GUI Control Plane

A lightweight Docker control plane for managing compose-based applications through a Go dashboard, BoltDB state, and host-managed `nginx` / `certbot`.

## Current operating model

- the dashboard is an internal-only controller
- host access is limited to `http://127.0.0.1:7000`
- default hardened runtime mode is `DASHBOARD_MODE=api`
- GUI is enabled only when needed and accessed through SSH tunneling
- managed applications keep running even if the dashboard container is removed or reinstalled

## Repository layout

```text
@GUIDocker/
├── dashboard/      # Go application, HTML views, static assets, tests
├── website/        # Example public site stack
├── .paas/          # Runner extensions and deployment documentation
├── Dockerfile      # Root dashboard image build
└── docker-compose.yml
```

## Quick local start

```bash
cd dashboard
go build -o dashboard .
PAAS_PORT=7000 PAAS_ADMIN_USER=admin PAAS_ADMIN_PASS=admin@123 ./dashboard
```

Open:

```text
http://localhost:7000
```

Disable auth for local UI iteration:

```bash
cd dashboard
DASHBOARD_AUTH_DISABLED=true air
```

## Internal-only production access

On the server the dashboard should stay bound to:

```text
http://127.0.0.1:7000
```

To use the GUI remotely:

```bash
ssh -L 7500:127.0.0.1:7000 root@<SERVER_HOST>
```

Then open:

```text
http://127.0.0.1:7500
```

## Deployment entry points

- [`.paas/README.md`](.paas/README.md): bootstrap and update flows for the dashboard runtime itself
- [`dashboard/README.md`](dashboard/README.md): application runtime, configuration, local development, and TLS behavior

## Managed app TLS

The dashboard settings page still owns platform-wide TLS options for managed application domains:

- `certbot_email`
- `certbot_enabled`
- `certbot_staging`
- `certbot_auto_renew`
- `certbot_terms_accepted`

These settings do not expose the dashboard itself publicly.
