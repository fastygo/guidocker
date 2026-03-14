# Stable Version Plan

## Goal

Ship a fully usable stable release for simple stacks before the `Project` / `Service` refactor. In this release, one managed `App` remains the main unit of management and the admin panel stays a separate deployment.

## Target Scenario

- Single-user standalone admin panel with root privileges
- The admin panel can run on a bare IP and dedicated port, with an optional domain
- Managed applications are simple Compose stacks handled as one `App`
- No subdomains are required in this stable release
- Each app may expose one primary public domain and one primary proxy target port
- `Nginx` owns public ingress on `80` and `443`
- `Certbot` handles TLS issuance and renewal

## Must-Have Scope

### 1. Stable App Lifecycle

- [x] Keep manual Compose and public Git repo import flows working
- [x] Keep create, deploy, stop, restart, logs, and delete stable for simple stacks
- [x] Preserve safe cleanup of the app stack directory and owned Docker resources
- [x] Keep Scanner as a read-only audit tool for unmanaged leftovers

### 2. Admin Panel Endpoint Settings

- [x] Add persistent settings for admin host IP, admin port, optional admin domain, and SSL mode
- [x] Keep the admin panel reachable by bare IP without requiring a domain
- [x] Keep the admin endpoint separate from application public routing

### 3. App Public Access Settings

- [x] Add app settings for one primary public domain
- [x] Add app settings for the internal target port that `Nginx` should proxy to
- [x] Keep `80` and `443` reserved for `Nginx` instead of direct app binding
- [x] Route domain traffic through platform-managed `Nginx` templates instead of manual per-app proxy editing

### 4. App Environment Settings

- [x] Add app settings for managed environment variables
- [x] Store environment data per app in a platform-owned format
- [x] Materialize managed environment variables into a generated `.env` file inside the app stack directory
- [x] Ensure manual Compose apps and imported repo apps can reuse the managed env file

### 5. TLS and Certificates

- [x] Integrate `Certbot` with `Nginx` for certificate issuance
- [x] Support certificate renewal
- [x] Keep wildcard certificates and automatic subdomain flows out of this release

### 6. Safety and Validation

- [x] Validate domain conflicts before applying routing changes
- [x] Validate target port conflicts and reserved admin port conflicts
- [x] Validate generated `Nginx` configuration before reload
- [x] Validate certificate issuance prerequisites before running `Certbot`
- [x] Fail early with clear operator-facing messages when routing or TLS cannot be applied

### 7. Safe Deletion Rules

- [x] Delete app containers, owned volumes, stack directory, and BoltDB record
- [x] Remove platform-generated `Nginx` config for the app on delete
- [x] Remove app-owned managed env files on delete
- [x] Remove app-owned certificates only when they are not shared
- [x] Leave external resources untouched and warn clearly when manual cleanup is required

### 8. Minimal Operational Visibility

- [x] Show public URL, proxy target port, SSL status, and env summary on the app detail page
- [x] Keep logs and runtime status visible in the current UI

## Out of Scope for This Stable Version

- `Project` / `Service` refactor
- Automatic subdomain assignment
- Wildcard SSL
- Multi-user / RBAC
- Background runtime-to-database sync
- Built-in CRON subsystem
- Complex big-stack orchestration such as service-by-service `Supabase` management

## Release Acceptance Criteria

- [x] The admin panel can run on a bare IP and custom port
- [x] A simple app can be created from Compose YAML or imported from a public repo
- [x] An app can be configured with env values, one public domain, and one proxy target port
- [x] `Nginx` routes the domain to the configured app port
- [x] `Certbot` can issue and renew a certificate for the app domain
- [x] Delete removes the app stack and all platform-owned routing and env artifacts without touching external resources