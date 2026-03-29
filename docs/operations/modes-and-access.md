# Modes And Access

## Why `DASHBOARD_MODE` Exists

The dashboard supports two runtime modes:

- `gui`
- `api`

This is a deliberate operational boundary, not just a UI toggle.

## `gui` Mode

In `gui` mode:

- the HTML operator interface is available,
- API routes are also available,
- browser-based administration works through direct internal access or an SSH tunnel.

Use this mode for:

- operator sessions,
- troubleshooting,
- configuration review,
- manual lifecycle actions.

## `api` Mode

In `api` mode:

- only `/api/*` routes are intended to be available,
- GUI routes return `404`,
- the runtime is better suited for automation-only or hardened environments.

Use this mode for:

- unattended environments,
- automation-first operation,
- reduced attack surface when the GUI is not needed.

## Switching Modes

A runtime deployed through the dashboard deployment flows commonly comes up in `api` mode.

To switch temporarily to GUI mode:

```bash
sed -i 's/DASHBOARD_MODE: api/DASHBOARD_MODE: gui/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

To return to API mode:

```bash
sed -i 's/DASHBOARD_MODE: gui/DASHBOARD_MODE: api/' /opt/${INPUT_APP_NAME}/docker-compose.yml
docker compose -p "${INPUT_APP_NAME}" -f "/opt/${INPUT_APP_NAME}/docker-compose.yml" up -d --remove-orphans
```

## SSH Tunnel Access

Recommended operator access pattern:

```bash
ssh -N -L 7500:127.0.0.1:7000 -i ~/.ssh/<KEY_FILE> root@<SERVER_HOST>
```

Then open:

```text
http://127.0.0.1:7500
```

Why this is recommended:

- the dashboard stays internal-only on the server,
- public ingress remains dedicated to managed apps,
- browser origin and cookie behavior remain stable,
- admin traffic is not exposed through public routing by default.

## Port Overrides

The dashboard runtime can be moved without changing stored app state.

Relevant variables:

- `PAAS_PORT`
- `SERVER_PORT`
- `SERVER_HOST`

Examples:

- `PAAS_PORT=7000` for the default internal controller port,
- `PAAS_PORT=7500` for a custom controller port,
- `SERVER_HOST=127.0.0.1` to bind locally.

Changing host or port does not by itself delete:

- platform settings,
- app definitions,
- compose stacks,
- certificates,
- nginx routes.

## Removing Or Hiding The Admin GUI

If the goal is to disable the operator UI while keeping managed applications alive, use `DASHBOARD_MODE=api`.

This preserves:

- dashboard API functionality,
- stored state,
- managed application runtime behavior.

In other words, removing GUI exposure is not the same as removing the managed platform state.
