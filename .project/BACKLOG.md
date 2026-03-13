# Backlog

## Done

- [x] App deletion (DELETE /api/apps/{id})
- [x] Safe self-removal scenario for the admin (maintenance mode for clean shutdown before removal)
- [x] Deploy and remove GUI while keeping the app running
- [x] Place GUI on top of existing manual apps and auto-discover them (Scanner)

## Planned

### Repo import (Create App from Git)

- [ ] Add `import_repo` mode alongside direct YAML input
- [ ] UI: tab/toggle in compose form — Repo URL, Branch (optional), Compose file path, Deploy now checkbox
- [ ] API: `POST /api/apps/import` with payload `{ name, repo_url, branch?, compose_path?, auto_deploy }`
- [ ] Service: clone repo to `STACKS_DIR/<app-id>`, read compose (or generate for static sites), save to BoltDB, optionally deploy
- [ ] Security: allow only `https://`, limit `git clone` timeout, handle private repos via secure backend param, clean temp dirs on error

### PaaS production epic (next phase)

- [ ] Full PaaS layer: apps lifecycle, SSL, nginx, CRON/cleanup, hardening
- [ ] Meet all production PaaS GUI requirements from the guide

## Known limitations (MVP scope)

- No automatic nginx config generation per app
- No wildcard SSL / Let's Encrypt
- No multi-user / RBAC
- No background sync of container state with stored apps

## Notes

**Port 80:** Multiple apps cannot all bind to 80. Use a reverse proxy (Nginx/Caddy) and assign internal ports per app.

**Current Create App flow:** Accepts only `name` and `compose_yaml`; repo URL is not supported yet (see Repo import above).
