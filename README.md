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
└── .project/
    └── www/             # Static landing page
        ├── index.html   # Product intro, feature highlights
        ├── style.input.css  # Tailwind source (shadcn tokens)
        └── style.css    # Compiled CSS (build output)
```

- **dashboard/** — Go server with HTTP Basic Auth, BoltDB, and Docker Compose integration. Serves the full admin UI (Overview, Apps, Compose, Logs, Scanner, Settings) and REST API.
- **.project/www/** — Static landing page (HTML + Tailwind) introducing the product. Uses shadcn-style design tokens. Built via `npm run build:www` from `dashboard/`.

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

# Landing page (.project/www)
npm run build:www
```

## Documentation

- [dashboard/README.md](dashboard/README.md) — Full dashboard docs: setup, API, deployment, Makefile targets.
