# Architecture Overview

## Purpose

The dashboard is an internal control plane for:

- managing application records and runtime state,
- rendering the operator GUI,
- exposing a machine-friendly JSON API,
- coordinating host `nginx` routing and `certbot` operations for managed apps.

The current design favors predictable operator behavior over browser-heavy client logic.

## Core Principles

- The GUI is server-rendered.
- Critical GUI actions use HTML `POST` forms.
- The JSON API remains available for automation.
- The dashboard runtime can be started in either `gui` or `api` mode.
- Managed applications keep running independently of whether the admin GUI is enabled.

## Why The GUI Is Autonomous

The runtime was intentionally refactored to reduce dependence on external services during ordinary operator work.

Autonomous here means:

- no required icon or UI CDN calls for core screens,
- page data is rendered on the server before HTML is returned,
- forms submit directly to the dashboard instead of relying on page-local fetch orchestration,
- operator workflows continue to work through an SSH tunnel to an internal address.

This reduces:

- blank pages caused by client-side data bootstrap failures,
- browser origin and cookie mismatches,
- surprises when external assets or third-party UI resources are unavailable.

## Why GUI Mutations Use POST

The GUI now uses standard browser form submission for settings and operational actions.

This was chosen because it provides:

- reliable browser semantics,
- clean Post-Redirect-Get behavior,
- server-owned validation,
- preserved form values on validation errors,
- less client-side state management.

The resulting behavior is simpler:

1. The browser submits a `POST`.
2. The server validates and performs the action.
3. The server redirects to the follow-up page.
4. The page renders the final state plus a flash message.

## API And GUI Relationship

The API and GUI are intentionally separate interfaces over the same domain logic.

- The GUI is optimized for human operators.
- The API is optimized for scripts, automation, and deployment flows.

This allows the project to:

- keep API compatibility for automation,
- harden unattended deployments by disabling the GUI,
- use the same underlying persistence and orchestration rules in both cases.

## Dashboard Runtime Modes

`DASHBOARD_MODE` controls which surface is exposed:

- `gui`: HTML pages plus API routes,
- `api`: API routes only.

This is not just a UI preference. It is the supported runtime boundary between:

- operator-facing sessions,
- automation-only or hardened environments.

## Independence Of Managed Applications

Managed applications do not depend on the admin GUI being enabled.

As long as persistent state and runtime artifacts remain in place:

- the app definitions remain in BoltDB,
- compose stacks remain on disk,
- host `nginx` configuration remains active,
- certificates remain on the host.

Because of that:

- changing the dashboard port does not delete app state,
- switching `gui` to `api` does not remove app state,
- removing GUI exposure does not stop published apps by itself.
