# Backlog

## Stable Baseline Delivered

These items are already delivered in the current pre-`Project` / pre-`Service` stable baseline and should not be re-planned as new work.

- [x] App deletion via `DELETE /api/apps/{id}`
- [x] Safe self-removal flow for the admin panel
- [x] Keep deployed apps running after removing the GUI
- [x] Read-only Scanner for discovered manual Docker resources
- [x] Public Git repo import with Compose-first flow and Dockerfile fallback
- [x] Persistent admin endpoint settings: host IP, port, optional domain, and TLS mode
- [x] Bare IP admin panel support without requiring a domain
- [x] One public domain and one proxy target port per app
- [x] Reserve `80` and `443` for platform ingress instead of direct app binding
- [x] Platform-managed `Nginx` routing for app domains
- [x] Managed environment variables persisted per app and materialized into `.platform.env`
- [x] `Certbot` certificate issuance for app domains
- [x] Certificate renewal path with host-managed execution
- [x] Domain conflict, proxy port conflict, and TLS prerequisite validation
- [x] Safe delete preflight for shared or external resources
- [x] Conditional certificate removal only when the certificate is not shared
- [x] App detail visibility for URL, proxy port, TLS status, env summary, logs, and runtime state

## Deferred From Stable Baseline

These items were intentionally left out of the first stable cutoff and belong to later planning.

- [ ] Optional admin-domain TLS with full routing and certificate reconciliation
- [ ] Basic health or reachability status for proxied apps
- [ ] In-place app updates that preserve domain and env settings
- [ ] Backup and restore for stateful apps
- [ ] Rich routing previews before save
- [ ] Extended scanner coverage for proxy and certificate leftovers
- [ ] Final production-hardening validation pass before declaring the stable baseline fully closed

## Immediate Priority

- [ ] Add explicit `Host port` and `Container port` support for Dockerfile fallback import instead of forcing one numeric `App port` value to generate `%d:%d`
  - Dockerfile fallback must collect `Host port` and `Container port` as separate values
  - Generated Compose must map ports as `HOST:CONTAINER` instead of forcing identical values
  - Validation must keep `80` and `443` reserved for platform ingress on the host side
  - The UI must clearly explain that `Host port` is for direct server access and `Container port` is what the app listens on internally
  - Existing imported apps must remain readable even if they were created with the legacy single-port model
  - Operators should be able to use common mappings such as `777:80`, `8080:3000`, or `9000:8080`

## Architecture Improvement Plan

These improvements are not blockers for the current stable baseline, but they should guide the next refactor so the platform stays flexible and scalable as it grows beyond the single-`App` model.

### Layer Boundaries

- [ ] Move pure business entities and domain rules into a stricter domain core with fewer mixed responsibilities
- [ ] Separate application contracts and orchestration interfaces from the pure domain layer where helpful
- [ ] Keep infrastructure details out of business-facing contracts unless they are part of an intentional port

### Application Services

- [ ] Split the current app lifecycle service into smaller focused services such as import, deploy, routing, env management, and cleanup coordination
- [ ] Introduce explicit orchestration services for cross-cutting operational flows instead of expanding one large service
- [ ] Keep lifecycle rules testable through narrow interfaces and isolated business scenarios

### Delivery Layer

- [ ] Separate HTML page handlers, JSON API handlers, and operational endpoints into clearer delivery modules
- [ ] Keep HTTP handlers thin and focused on transport concerns: parsing, validation, response mapping, and auth boundaries
- [ ] Avoid letting delivery code call infrastructure operations directly when an application service can coordinate the flow

### Infrastructure and Composition

- [ ] Keep composition centralized in the application entrypoint so dependencies remain explicit and replaceable
- [ ] Continue isolating Docker, Git, BoltDB, Nginx, and Certbot behind stable interfaces
- [ ] Add clearer boundaries between platform runtime adapters and business orchestration logic

### Growth Readiness

- [ ] Refactor around the future `Project` / `Service` model without breaking the stable single-`App` baseline
- [ ] Preserve the current admin panel as a valid reference implementation of a pragmatic layered backend
- [ ] Keep the architecture suitable as a teaching case for layered backend design, even if it is not a fully academic clean-architecture example

## Product Principles

- Single-user standalone admin panel with root privileges
- One `Project` is one deployable Docker Compose stack
- `Services` are child resources inside a `Project`
- Deploy, stop, restart, and delete stay at the `Project` level
- The admin panel endpoint is separate from public project routing
- The admin panel must work on a bare IP without requiring a domain
- Public websites and APIs should run as independent projects behind the platform proxy layer
- Scanner stays read-only for unmanaged resources and outputs manual cleanup commands
- CRON is not part of the admin panel core and should run as a separate managed project when needed

## Immediate Next Milestone

- [ ] Deliver the first `Project` / `Service` backend model while preserving compatibility with existing `App` records
- [ ] Keep lifecycle and deletion semantics safe at the project level before adding more networking and TLS automation
- [ ] Introduce the first project details view with read-only services so the new model is visible before deeper routing work starts

## Priority Roadmap

### Phase 1. Project Model and Safe Lifecycle

#### Step 1. Project and Service Data Model

- [ ] Introduce `Project` as the main managed resource for complex Docker Compose stacks
- [ ] Define `Service` as a child runtime resource inside a project
- [ ] Store project source metadata, compose path, stack directory, and cleanup ownership in the data model
- [ ] Keep backward compatibility with existing app records during the transition to the project model

#### Step 2. Backend Lifecycle and Orchestration

- [ ] Keep deploy, stop, restart, delete, and cleanup operations at the project level
- [ ] Resolve runtime operations by project identity instead of by individual service
- [ ] Aggregate project status from service state instead of treating one container as the whole app
- [ ] Keep service-level actions read-only in the initial phase, except for inspection and logs

#### Step 3. Project and Service UI

- [ ] Replace the app-centric primary view with a project-centric primary view
- [ ] Add a project details page with a read-only services list
- [ ] Show per-service status, logs, image, ports, mounts, and health inside the project view
- [ ] Add a clear project summary for source, storage, exposure, and runtime state

#### Step 4. Safe Deletion and Cleanup Rules

- [ ] Add a deletion preview for project-owned containers, networks, volumes, and stack directories
- [ ] Define strict cleanup rules for project-owned resources versus external resources
- [ ] Warn when a project uses external volumes, external networks, or bind mounts outside the managed stacks directory
- [ ] Keep Scanner as the fallback tool for unmanaged leftovers and manual cleanup guidance

### Phase 2. Safe Networking and Public Exposure

#### Step 1. Network and Exposure Data Model

- [ ] Add a dedicated admin endpoint configuration model: host IP, public port, optional domain, and SSL mode
- [ ] Add a project exposure model: internal service port, public hostname, protocol, and exposure mode
- [ ] Add a platform setting for the base domain used to generate default subdomains
- [ ] Keep the admin endpoint model separate from public project routing

#### Step 2. Settings and Project UI

- [ ] Add settings UI for the admin endpoint so the panel can run on a bare IP or on an optional domain
- [ ] Add settings UI for the platform base domain and certificate mode
- [ ] Add project-level UI for public hostname assignment
- [ ] Allow automatic subdomain generation from the configured base domain
- [ ] Allow operators to replace the generated hostname with a manual hostname override

#### Step 3. Reverse Proxy and Routing

- [ ] Use `Nginx` as the first reverse proxy implementation
- [ ] Add a reverse proxy layer for all public traffic
- [ ] Stop binding all applications directly to host port `80`
- [ ] Assign internal ports per project and expose only approved entry points
- [ ] Treat public websites and APIs as independent projects that can be published on `80` and `443`
- [ ] Keep proxy management controlled by the platform model instead of free-form per-app Nginx editing
- [ ] Add deterministic routing rules for admin traffic versus public project traffic

#### Step 4. Certificates and TLS

- [ ] Add `Certbot`-managed certificate issuance for domains and subdomains served by `Nginx`
- [ ] Add automatic certificate renewal flow
- [ ] Add fixed domain support for production use without requiring wildcard certificates
- [ ] Support project-level TLS for both generated subdomains and manually assigned hostnames

#### Step 5. Access Policy and Safety Checks

- [ ] Allow the admin panel to run on a bare IP and custom port without requiring a domain
- [ ] When a public hostname is configured for a project, block or redirect direct IP access for that project
- [ ] Add validation for hostname conflicts, duplicate routes, and reserved admin routes
- [ ] Add validation for port conflicts before applying proxy changes
- [ ] Add preview and rollback support for routing configuration updates

### Phase 3. Production Readiness for Big Stacks

#### Step 1. Preflight Validation

- [ ] Add preflight validation before import or deploy
- [ ] Validate port conflicts, missing files, invalid compose settings, and required env files
- [ ] Detect external volumes, external networks, and bind mounts outside the stack directory
- [ ] Fail early with a clear project-level validation report before runtime changes are applied

#### Step 2. Health, Readiness, and Dependencies

- [ ] Add aggregated project health from all managed services
- [ ] Add per-service readiness and health checks for HTTP, TCP, and container status
- [ ] Model dependency visibility for databases, Redis, workers, and gateways inside a project
- [ ] Surface startup ordering and failed dependency states in the project view

#### Step 3. Secrets and Stateful Storage

- [ ] Add secrets and `.env` management per project
- [ ] Define safe handling rules for stateful services such as PostgreSQL and Redis
- [ ] Separate managed project data from external storage dependencies
- [ ] Show clear warnings when a project depends on external state that will not be removed automatically

#### Step 4. Backup and Recovery

- [ ] Add backup flows for stateful services such as PostgreSQL and Redis
- [ ] Add restore flows for project-owned stateful data
- [ ] Define retention, naming, and storage rules for backup artifacts
- [ ] Add recovery guidance and validation after restore

#### Step 5. Updates, Migrations, and Rollback

- [ ] Add update flow based on stored source metadata and resolved commit
- [ ] Add staged rollback support when an updated project fails health or readiness checks
- [ ] Add migration hooks for complex stacks such as `n8n` and `Supabase`
- [ ] Add dependency-aware upgrade guidance for multi-service projects

#### Step 6. Operational Guardrails

- [ ] Add log retention and export policy for production troubleshooting
- [ ] Add runtime guardrails for dangerous compose options and unsafe exposure patterns
- [ ] Add safer defaults for production deployment profiles where possible
- [ ] Add operator-facing warnings before high-risk actions on stateful or externally connected projects

### Phase 4. UX and Operational Polish

#### Step 1. Scanner and Inventory UX

- [ ] Make Scanner output project-aware for orphan runtime resources, orphan directories, and stale admin instances
- [ ] Group scanner findings by managed project, unmanaged runtime, and unmanaged filesystem resources
- [ ] Improve manual cleanup command visibility and copy-ready formatting

#### Step 2. Project Operations UX

- [ ] Add project-level operational summaries for ports, storage, logs, and exposure
- [ ] Improve navigation between project summary, services, logs, health, and cleanup preview
- [ ] Add clearer confirmations and impact summaries for destructive actions

#### Step 3. Production UX Alignment

- [ ] Add clearer warnings and cleanup guidance in the UI
- [ ] Align the main project workflows with the production PaaS GUI requirements from the guide
- [ ] Add concise operator help text for routing, certificates, backups, and rollback flows

## Constraints and Out of Scope

- No multi-user / RBAC
- No background sync that continuously mutates stored app state from runtime state
- No wildcard SSL requirement in the initial production phase
- No built-in CRON subsystem in the admin panel core