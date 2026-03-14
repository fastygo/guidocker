# Smart GUI Control Plane

A lightweight PaaS for managing Docker applications. Deploy, monitor, and clean up compose-based stacks from a single control surface.

## Architecture

```text
@GUIDocker/
├── dashboard/           # Go application — main PaaS server
│   ├── config/          # Configuration
│   ├── domain/          # Entities, ports
│   ├── infrastructure/  # BoltDB, Docker CLI
│   ├── interfaces/      # HTTP handlers, middleware
│   ├── usecase/         # App lifecycle, scanner
│   ├── views/           # HTML templates
│   └── static/          # Tailwind CSS, assets
│
├── website/             # Public home page / landing site
│   ├── docker-compose.yml  # Importable static site stack
│   ├── index.html       # Product intro, feature highlights
│   ├── style.input.css  # Tailwind source
│   └── style.css        # Compiled CSS (build output)
│
└── .project/            # Project planning notes and internal docs
```

- **dashboard/** — Go server with HTTP Basic Auth, BoltDB, and Docker Compose integration. Serves the full admin UI (Overview, Apps, Compose, Logs, Scanner, Settings) and REST API.
- **website/** — Static landing page (HTML + Tailwind) for the public product home page. Built via `npm run build:www` from `dashboard/` and importable into the admin panel via `website/docker-compose.yml`.

## Quick Start

```bash
cd dashboard
go build -o dashboard .
PAAS_ADMIN_USER=admin PAAS_ADMIN_PASS=admin@123 ./dashboard
```

Open `http://localhost:3000`.

## Build Frontend Assets

```bash
cd dashboard
npm install

# Dashboard UI (Tailwind)
npm run build:css

# Landing page (website/)
npm run build:www
```

## Deployment Model

- The dashboard remains an autonomous temporary GUI for a single root operator.
- Applications are deployed by Docker Compose and continue running after the dashboard container is removed.
- Public routing is handled by host-installed `nginx` and `certbot`, managed by the dashboard.
- Applications are attached to the managed Docker network `paas-network` and should be routed by internal container port, not by published host ports.

## Website Import Checklist

Use this flow to deploy the official landing page through the admin panel:

1. Import from Git.
2. Use repository URL `https://github.com/fastygo/guidocker.git`.
3. Set compose file path to `website/docker-compose.yml`.
4. Leave app port empty because the compose file already defines the service.
5. Deploy the imported app.
6. Open app settings in the dashboard.
7. Set `ProxyTargetPort` to `80` because routing now targets the internal container port on `paas-network`.
8. Save the app configuration.
9. Add the public domain later through routing/settings in the admin panel.
10. Enable HTTPS only after DNS is ready and host certbot settings are configured.

## Documentation

- [dashboard/README.md](dashboard/README.md) — Full dashboard docs: setup, API, deployment, Makefile targets.
