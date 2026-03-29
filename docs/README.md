# Documentation

This folder contains the operator and architecture documentation for the dashboard runtime and managed application flows.

## Structure

### Architecture

- [`architecture/overview.md`](./architecture/overview.md): architecture goals, SSR and POST rationale, autonomous GUI behavior, `api|gui` mode intent, and persistence boundaries.

### Deployment

- [`deployment/dashboard-runtime.md`](./deployment/dashboard-runtime.md): `bootstrap-direct`, `deploy-direct`, `deploy`, dry-run behavior, GUI mode after deployment, and runtime update rules.
- [`deployment/managed-apps.md`](./deployment/managed-apps.md): managed application deployment flows and the separation between platform settings and app routing.

### Operations

- [`operations/modes-and-access.md`](./operations/modes-and-access.md): `DASHBOARD_MODE`, SSH tunnel access, port overrides, and disabling the GUI without deleting state.
- [`operations/settings-and-persistence.md`](./operations/settings-and-persistence.md): difference between `.paas/config.yml` inputs and runtime database settings, plus the recommended seed-once model.
- [`operations/nginx-certbot.md`](./operations/nginx-certbot.md): nginx and certbot responsibilities, deployment scenarios, and TLS prerequisites.

### Runbooks

- [`runbooks/dashboard-operations.md`](./runbooks/dashboard-operations.md): production-oriented steps for bootstrap, deploy, GUI enablement, health verification, and missing settings recovery.
- [`runbooks/managed-app-operations.md`](./runbooks/managed-app-operations.md): production steps for creating, updating, and validating managed applications.
- [`runbooks/incident-response.md`](./runbooks/incident-response.md): fast incident handling for GUI 404s, empty settings, certbot failures, unreachable domains, and tunnel-related operator problems.

### FAQ

- [`faq/operator-faq.md`](./faq/operator-faq.md): concise answers to recurring operator questions about modes, settings, deploy behavior, persistence, and TLS.

### Troubleshooting

- [`troubleshooting/common-issues.md`](./troubleshooting/common-issues.md): common failures, quick checks, empty settings pages, API-vs-GUI confusion, and debugging commands.

## Recommended Reading Order

1. Start with [`architecture/overview.md`](./architecture/overview.md).
2. Read [`deployment/dashboard-runtime.md`](./deployment/dashboard-runtime.md) if you operate the dashboard itself.
3. Read [`operations/settings-and-persistence.md`](./operations/settings-and-persistence.md) before changing deployment flows that can write runtime settings.
4. Use [`runbooks/dashboard-operations.md`](./runbooks/dashboard-operations.md) and [`runbooks/managed-app-operations.md`](./runbooks/managed-app-operations.md) for production actions.
5. Use [`runbooks/incident-response.md`](./runbooks/incident-response.md) during live incidents.
6. Use [`faq/operator-faq.md`](./faq/operator-faq.md) and [`troubleshooting/common-issues.md`](./troubleshooting/common-issues.md) during handoffs and root-cause analysis.