Ниже готовая документация на английском языке, с учётом всего сделанного рефакторинга (SSR+POST для GUI, API сохранён, внешний контент для UI убран, `mode=api|gui`, стабильность через туннель и независимость приложений).

# PaaS Dashboard Architecture & Operations Guide

## Purpose and current design

This guide describes the current operation model after the hybrid refactor:
- all key GUI pages are server-rendered (SSR),
- all user mutations in GUI are `POST` form submissions with PRG redirects,
- JSON API is preserved for automation and integration,
- dashboard runtime can work in `gui` or `api` mode,
- admin UI is safe to run through SSH tunnel and can be fully disabled without touching managed apps state.

Goal: stable operator UX without cross-origin / cookie mismatches and predictable infrastructure behavior.

## Why the system is now autonomous

The GUI is intentionally designed to be autonomous at runtime:

- No dependency on external CSS/icon/CDN endpoints for core UI rendering.
- All admin templates and static files are served from the dashboard container/binary itself (including local SVG icons).
- Page data is rendered by backend handlers before HTML is sent, so admin pages do not require pre-load client fetches from external/API endpoints.
- All operational mutations are done via standard browser form submits, not API fetch logic in page scripts.
- The system still supports API automation separately.

This reduces:
- runtime coupling to network/CDN availability,
- initial blank states caused by blocked JS requests,
- hidden failures when browser origin/cookie behavior differs between local/remote access methods.

## Why POST is used for GUI actions now

All critical GUI actions were moved to HTML `POST` routes:

- app config save,
- platform settings save,
- certbot renew trigger,
- app deploy/restart/stop,
- delete app (with confirmation page),
- compose create/update/import flows.

This decision solves several issues:

- `POST` + `303` redirect keeps browser behavior deterministic.
- Flash messages are passed via query string on redirect (`msg`, `err`) and shown after page reload.
- No manual JSON parsing/rendering cycles on the page for every action.
- No fragile front-end state handling or race conditions from concurrent fetches.
- Works consistently through browser tunnels and internal addresses.

Validation errors re-render the same page with original form values, preventing user data loss on failed submissions.

## Why DASHBOARD_MODE exists (api|gui)

`DASHBOARD_MODE` explicitly splits capabilities:

- `gui` mode:
  - full web interface enabled (`/`, `/apps`, `/apps/{id}`, `/settings`, compose pages),
  - API routes still available.
  - suitable for operator sessions and diagnostics.

- `api` mode:
  - HTML UI routes are disabled (404 if accessed),
  - only API routes are registered,
  - intended for hardened runtime and machine-driven use.

This is the supported switch for “disable/remove admin UI” without destroying managed application state.

## Mode behavior in practice

- `DASHBOARD_MODE=gui`:
  - operator can use browser session (including via SSH tunnel),
  - form submissions keep app state updates reliable.
- `DASHBOARD_MODE=api`:
  - dashboard can still execute all orchestration logic through `/api/*`,
  - browser access to `/` and `/apps` is not expected.

Both modes use the same core services and persistence:
- same app repository and DB,
- same host managers (`nginx`, `certbot`),
- same deployment logic.

## Port override and “remove admin” without data loss

Runtime port and host are fully configurable:

- `PAAS_PORT` (primary),
- `SERVER_PORT` (legacy),
- `SERVER_HOST` (bind address).

Examples:
- `SERVER_HOST=127.0.0.1`, `PAAS_PORT=7000` keeps dashboard local/private.
- `PAAS_PORT=7500` fully overrides listener port.
- `DASHBOARD_MODE=api` removes the GUI surface while preserving all management API behavior.

Important: changing mode or port does not erase settings if data paths remain unchanged.

## What happens to settings and app data when UI changes

Data persistence is independent from UI mode:

- app metadata and config live in BoltDB (`BOLT_DB_FILE`, default `/opt/stacks/.paas.db`),
- compose stacks and runtime files live under `STACKS_DIR`,
- settings can be changed in GUI or API and are written to the same domain services,
- each app has own lifecycle and routing artifacts; one app operation should not reset another app’s config.

So:
- switching `DASHBOARD_MODE`,
- restarting dashboard container,
- changing external exposure,
- or using only API mode

does not delete/rewrite managed app definitions by itself.

## Stable operation with browser + API + SSH tunnel

Recommended secure operator access model:

- run dashboard on internal address (default `127.0.0.1:7000`),
- open only SSH tunnel:
  - `ssh -L 7500:127.0.0.1:7000 root@<host>`,
- open `http://127.0.0.1:7500` in browser,
- API clients still call remote API host directly or via the same tunnel to `/api/*`.

Benefits:
- avoids public exposure of admin endpoint,
- avoids cookie/domain mismatch that caused earlier empty/blocked behavior in mixed origin setups,
- keeps managed app public endpoints unchanged.

## Nginx and certbot deployment scenarios

### Scenario 1: Internal admin + public app routing
- dashboard only internal (tunnel),
- `nginx` on host serves managed domains and points to app containers as configured,
- certbot automates TLS for each public domain when enabled per app.

### Scenario 2: GUI on public port through nginx
- proxy `/` or `/admin` to dashboard only if explicitly needed,
- keep auth enabled and TLS on admin path if exposed,
- still safe because GUI now SSR and form-based.

### Scenario 3: API-only runtime
- set `DASHBOARD_MODE=api`,
- do not expose GUI endpoints,
- use API calls from automation or scripts.

### Certbot workflow notes
- app-level `PublicDomain` + `UseTLS` must be configured,
- app route must resolve to host,
- HTTP challenge path must be reachable before renew/enforce HTTPS,
- use staging mode for first-time verification safety if needed.

Example commands:
```bash
sudo nginx -t
sudo systemctl reload nginx
sudo certbot certificates
sudo systemctl status certbot.timer
```

## Frequent errors and troubleshooting

## FAQ

1) Why does cert refresh fail or not appear in UI?
- check API/GUI routing mismatch and ensure dashboard mode allows GUI or API as expected,
- verify dashboard credentials and session,
- verify app has public domain and `UseTLS` setting saved,
- check `nginx -t` and DNS reachability before cert request.

2) Why did the public domain disappear from app details?
- ensure app config was saved after `POST /apps/{id}/config`,
- confirm `appUseCase` returns updated DB state (not stale client-side cache),
- if changed through scripts, verify persistence file exists and is writable.

3) Can I switch from GUI mode to API mode?
- yes. Restart with `DASHBOARD_MODE=api`.
- API continues to work; UI routes intentionally stop.

4) Can I keep apps working after deleting dashboard UI?
- yes. If `STACKS_DIR` and DB remain, managed apps and routing artifacts are unchanged.

5) Is local import from git still possible without GUI?
- yes via API endpoint(s) using the same application service methods.

6) Is there any blocking external resource left?
- no required UI dependency on `unpkg`/CDN remains for core UI after icon local migration,
- external dependencies still include external git repos (import mode), docker registry, ACME and DNS challenge paths — these are operational dependencies, not GUI assets.

## Debugging checklist (operational)

- Check runtime mode:
  - inspect envs and logs for `DASHBOARD_MODE`.
- Verify server is alive:
  - `curl -u admin:pass http://127.0.0.1:7000/api/health`.
- Verify routes:
  - in gui mode: `curl -I http://127.0.0.1:7000/login`
  - in api mode: `/login` expected not available.
- Verify DB and stack persistence:
  - check `BOLT_DB_FILE` and `STACKS_DIR` mount ownership/permissions.
- Verify nginx mapping:
  - `sudo nginx -t`, `sudo tail -f /var/log/nginx/error.log`.
- Verify certbot state:
  - `sudo certbot certificates`, `sudo ls -l /etc/letsencrypt/live`.

## Security and production notes

- keep dashboard behind SSH tunnel or internal network,
- use strong `PAAS_ADMIN_USER/PAAS_ADMIN_PASS`,
- disable auth only for controlled local development (`DASHBOARD_AUTH_DISABLED=true`),
- prefer `DASHBOARD_MODE=api` for unattended environments,
- never mount DB path writable by unrelated services.

## Recommended baseline for your current setup

1. Keep dashboard in `gui` mode for operator access.
2. Bind to loopback / private host.
3. Use SSH tunnel in daily workflows.
4. Keep `DASHBOARD_MODE=api` on nodes where only API jobs are required.
5. Use local asset strategy and avoid CDN reliance for stability.

If you want, I can now convert this into a committed `dashboard/ARCHITECTURE.md` (or `dashboard/docs/OPERATING.md`) file with the same sections plus a short deployment matrix and runbook templates.