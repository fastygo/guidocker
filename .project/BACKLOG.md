# Backlog

## Done

- [x] App deletion via `DELETE /api/apps/{id}`
- [x] Safe self-removal flow for the admin panel
- [x] Keep deployed apps running after removing the GUI
- [x] Read-only Scanner for discovered manual Docker resources
- [x] Public Git repo import with Compose-first flow and Dockerfile fallback

## Product Principles

- Single-user standalone admin panel with root privileges
- One `Project` is one deployable Docker Compose stack
- `Services` are child resources inside a `Project`
- Deploy, stop, restart, and delete stay at the `Project` level
- Scanner stays read-only for unmanaged resources and outputs manual cleanup commands
- CRON is not part of the admin panel core and should run as a separate managed project when needed

## Priority Roadmap

### Phase 1. Project Model and Safe Lifecycle

- [ ] Introduce `Project` as the main managed resource for complex stacks
- [ ] Treat containers, databases, Redis, workers, gateways, and sidecars as `Services` inside a project
- [ ] Keep lifecycle operations at the project level: deploy, stop, restart, delete, cleanup
- [ ] Add a project details view with a read-only services list
- [ ] Show per-service status, logs, image, ports, and health inside the project view
- [ ] Add a deletion preview for project-owned containers, networks, volumes, and stack directories
- [ ] Define strict cleanup rules for project-owned resources versus external resources

### Phase 2. Safe Networking and Public Exposure

- [ ] Add a reverse proxy layer for public traffic
- [ ] Stop binding all applications directly to host port `80`
- [ ] Assign internal ports per project and expose only approved entry points
- [ ] Add settings for admin panel URL, homepage URL, host IP, public port, and SSL mode
- [ ] When a public URL is configured, block or redirect direct IP access
- [ ] Keep proxy management controlled by the platform model instead of free-form per-app Nginx editing
- [ ] Add fixed domain and certificate support for production use

### Phase 3. Production Readiness for Big Stacks

- [ ] Add preflight validation before import or deploy
- [ ] Validate port conflicts, missing files, invalid compose settings, and required env files
- [ ] Detect external volumes, external networks, and bind mounts outside the stack directory
- [ ] Add aggregated project health and per-service readiness checks
- [ ] Add secrets and `.env` management per project
- [ ] Add backup and restore flows for stateful services such as PostgreSQL and Redis
- [ ] Add update and rollback flow based on stored source metadata and resolved commit
- [ ] Add migration and dependency hooks for complex stacks such as `n8n` and `Supabase`
- [ ] Add log retention and export policy for production troubleshooting
- [ ] Add runtime guardrails for dangerous compose options and unsafe exposure patterns

### Phase 4. UX and Operational Polish

- [ ] Make Scanner output project-aware for orphan runtime resources, orphan directories, and stale admin instances
- [ ] Add clearer warnings and cleanup guidance in the UI
- [ ] Add project-level operational summaries for ports, storage, logs, and exposure
- [ ] Meet the production PaaS GUI requirements from the guide

## Constraints and Out of Scope

- No multi-user / RBAC
- No background sync that continuously mutates stored app state from runtime state
- No wildcard SSL / Let's Encrypt in the initial production phase
- No built-in CRON subsystem in the admin panel core