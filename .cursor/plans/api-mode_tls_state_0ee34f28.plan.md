---
name: api-mode tls state
overview: Дополнить план API-only режима так, чтобы certbot/TLS состояние сохранялось в BoltDB и API-режим мог безопасно включать TLS для домена без зависимости от GUI и без лишних вызовов Let's Encrypt.
todos:
  - id: persist-platform-tls-state
    content: Закрепить в плане, что certbot/admin TLS settings сохраняются и читаются только через platform_settings в BoltDB.
    status: pending
  - id: apionly-settings-access
    content: Оставить /api/settings доступным в API-only mode как основной интерфейс восстановления TLS state.
    status: pending
  - id: idempotent-cert-check
    content: Добавить в план проверку существующих cert files перед вызовом certbot, предпочтительно внутри CertbotManager.EnsureCertificate().
    status: pending
  - id: preserve-tls-validation
    content: "Сохранить в app service обязательные prerequisites: certbot_enabled, certbot_email, certbot_terms_accepted."
    status: pending
  - id: separate-admin-vs-app-tls
    content: Явно ограничить текущий рефакторинг app-domain TLS логикой и не смешивать его с полноценным admin endpoint TLS.
    status: pending
  - id: add-targeted-tests
    content: Добавить тесты на persistence platform settings, API-only /api/settings и skip-certbot при уже существующем сертификате.
    status: pending
isProject: false
---

# API-Only Mode: TLS State And Certbot Persistence

## Goal

Сохранить certbot/TLS-настройки в persistent `BoltDB` и сделать поведение API-режима предсказуемым: GUI может отсутствовать, но `/api/settings` и `/api/apps/{id}/config` должны иметь достаточно данных, чтобы безопасно включать `UseTLS=true` для домена.

## What To Add To Existing Refactor Plan

### 1. Treat platform TLS settings as first-class persisted state

Use the existing persisted model in [E:/_@Go/@GUIDocker/dashboard/domain/platform_settings.go](E:/_@Go/@GUIDocker/dashboard/domain/platform_settings.go), stored through [E:/_@Go/@GUIDocker/dashboard/infrastructure/bolt/platform_settings_repo.go](E:/_@Go/@GUIDocker/dashboard/infrastructure/bolt/platform_settings_repo.go).

Keep these fields authoritative in `BoltDB`:

- `certbot_email`
- `certbot_enabled`
- `certbot_staging`
- `certbot_auto_renew`
- `certbot_terms_accepted`
- `admin_domain`
- `admin_use_tls`

Important: document that loss of these settings after redeploy means the underlying `BOLT_DB_FILE` volume/path was not persisted, not that GUI state was special.

### 2. Make API mode fully manageable without GUI

The existing API endpoint [E:/_@Go/@GUIDocker/dashboard/interfaces/paas_handlers.go](E:/_@Go/@GUIDocker/dashboard/interfaces/paas_handlers.go) already exposes `GET/PUT /api/settings`; keep it available in both GUI and API-only mode.

Ensure the API-only plan explicitly depends on:

- `RegisterAPIRoutes()` including `/api/settings`
- `/api/settings` being the only required interface for restoring certbot state
- startup continuing to load platform settings from the same `BoltDB` file in `main.go`

### 3. Prevent unnecessary certbot issuance when cert already exists

Current behavior in [E:/_@Go/@GUIDocker/dashboard/usecase/app/service.go](E:/_@Go/@GUIDocker/dashboard/usecase/app/service.go) is:

- `UpdateAppConfig(... UseTLS=true ...)`
- `reconcileRouting()`
- unconditional `certManager.EnsureCertificate(...)`

And [E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/certbot_manager.go](E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/certbot_manager.go) always builds a `certbot certonly --nginx ...` command.

Add a guard before invoking `certbot`:

- check whether `/etc/letsencrypt/live/<domain>/fullchain.pem` and `privkey.pem` already exist
- if both exist, skip `certbot` issuance and only re-apply nginx routing
- if missing, run the current issuance flow

This can live either:

- inside `CertbotManager.EnsureCertificate()` as an idempotency check, which is the safest option, or
- in app reconcile logic before calling the cert manager

Preferred plan choice: put the check in `CertbotManager`, so all callers get the same safe behavior.

### 4. Keep TLS prerequisites tied to persisted settings

Retain the existing validation in [E:/_@Go/@GUIDocker/dashboard/usecase/app/service.go](E:/_@Go/@GUIDocker/dashboard/usecase/app/service.go):

- `CertbotEnabled`
- `CertbotEmail`
- `CertbotTermsAccepted`

This is the correct answer to: “where will API mode get email if GUI is absent?”
It must come from `platform_settings` in `BoltDB`, not from current form state and not from parsing host certbot config at request time.

### 5. Clarify admin TLS versus app TLS

Today `admin_use_tls` and `admin_domain` are persisted, but real nginx/certbot issuance is implemented only for app domains, not for the dashboard endpoint itself.

Update the plan to make this explicit and choose one behavior:

- minimum-scope option: keep persisting `admin_*` fields but document that automatic TLS issuance currently applies only to managed app domains
- fuller option: add a separate admin-endpoint routing/certificate reconcile flow

Preferred scope for this refactor: minimum-scope only. Do not mix admin-endpoint reverse-proxy automation into the API-only refactor unless explicitly requested.

### 6. Add focused tests for the new certificate behavior

Extend tests near:

- [E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/manager_test.go](E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/manager_test.go)
- [E:/_@Go/@GUIDocker/dashboard/usecase/app/service_test.go](E:/_@Go/@GUIDocker/dashboard/usecase/app/service_test.go)
- [E:/_@Go/@GUIDocker/dashboard/usecase/settings/service_test.go](E:/_@Go/@GUIDocker/dashboard/usecase/settings/service_test.go)

Add coverage for:

- persisted certbot settings surviving service restart when the same `BoltDB` file is reused
- `UseTLS=true` failing when `certbot_email` / `certbot_enabled` / `terms_accepted` are absent
- `EnsureCertificate()` skipping issuance when cert files already exist
- nginx config enabling TLS immediately when cert files already exist
- API-only route set still exposing `/api/settings`

## Suggested Flow

```mermaid
flowchart LR
    apiSettings[/api/settings PUT/] --> boltState[(platform_settings in BoltDB)]
    boltState --> appConfig[/api/apps/{id}/config PUT/]
    appConfig --> tlsValidation[validateTLSPrerequisites]
    tlsValidation --> reconcileRouting[reconcileRouting]
    reconcileRouting --> certCheck{certFilesExist}
    certCheck -->|yes| applyNginx[apply nginx TLS config]
    certCheck -->|no| runCertbot[run certbot certonly]
    runCertbot --> applyNginx
```



## Key Files

- [E:/_@Go/@GUIDocker/dashboard/interfaces/routes.go](E:/_@Go/@GUIDocker/dashboard/interfaces/routes.go)
- [E:/_@Go/@GUIDocker/dashboard/interfaces/paas_handlers.go](E:/_@Go/@GUIDocker/dashboard/interfaces/paas_handlers.go)
- [E:/_@Go/@GUIDocker/dashboard/usecase/settings/service.go](E:/_@Go/@GUIDocker/dashboard/usecase/settings/service.go)
- [E:/_@Go/@GUIDocker/dashboard/usecase/app/service.go](E:/_@Go/@GUIDocker/dashboard/usecase/app/service.go)
- [E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/certbot_manager.go](E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/certbot_manager.go)
- [E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/nginx_manager.go](E:/_@Go/@GUIDocker/dashboard/infrastructure/hosting/nginx_manager.go)
- [E:/_@Go/@GUIDocker/dashboard/main.go](E:/_@Go/@GUIDocker/dashboard/main.go)

