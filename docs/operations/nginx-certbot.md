# Nginx And Certbot Operations

## Scope

The dashboard does not terminate public traffic for itself through host `nginx` by default.

Instead:

- the dashboard usually stays internal-only,
- host `nginx` publishes managed applications,
- host `certbot` manages certificates for managed application domains.

## Deployment Scenarios

### Scenario 1: Internal Dashboard, Public Managed Apps

This is the recommended model.

- dashboard listens on an internal port such as `127.0.0.1:7000`,
- operator GUI is accessed through an SSH tunnel,
- public domains point only to managed applications through host `nginx`.

### Scenario 2: Temporarily Exposed Dashboard GUI

This can be done, but it is not the default recommendation.

If used:

- keep authentication enabled,
- protect the path with TLS,
- keep exposure narrow and intentional.

### Scenario 3: API-Only Controller

Use this when the dashboard should remain operational but should not expose the GUI.

## TLS Prerequisites For Managed Apps

For certificate issuance to work, all of the following must be true:

- platform certbot settings are present,
- the app has a public domain configured,
- the app routing points to the correct target port,
- DNS resolves the public domain to the host,
- host ports 80 and 443 are correctly handled by `nginx`,
- ACME HTTP challenge requests can reach the host.

## Platform Settings Dependency

Per-app TLS intent is not enough by itself.

The runtime also needs valid platform settings such as:

- certbot email,
- automation enabled state,
- staging choice,
- renewal policy.

Those are platform-level settings, not per-app settings.

## Common Validation Commands

Run these on the server:

```bash
sudo nginx -t
sudo systemctl reload nginx
sudo certbot certificates
sudo systemctl status certbot.timer
```

## Common Failure Patterns

### Domain is configured but certificate is not issued

Check:

- platform settings actually exist in the dashboard database,
- the app is reachable over plain HTTP first,
- DNS and nginx routing are correct,
- certbot has permission and expected host directories mounted.

### GUI checkbox changed but runtime behavior did not improve

A GUI checkbox alone does not fix missing prerequisites.

The full path must still be valid:

- saved app config,
- saved platform settings,
- reachable public HTTP route,
- successful host `nginx` reload,
- successful `certbot` run.

## Dashboard Independence

Certificates and nginx configuration belong to the host-side runtime model for managed apps.

This means:

- disabling the GUI does not automatically remove certificates,
- changing dashboard mode does not automatically remove nginx routes,
- managed application ingress can continue to work even if the dashboard GUI is not exposed.
