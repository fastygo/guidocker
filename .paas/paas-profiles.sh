#!/usr/bin/env bash

# Example presets for the internal dashboard runtime.
# Replace placeholders before using any profile.

set -euo pipefail

PAAS_BIN="${PAAS_BIN:-$(cd "$(dirname "$0")" && pwd)/../paas.exe}"

run_paas() {
  local extension=$1

  env -i \
    HOME="$HOME" \
    USERPROFILE="${USERPROFILE:-$HOME}" \
    HOMEDRIVE="${HOMEDRIVE:-C:}" \
    HOMEPATH="${HOMEPATH:-/}" \
    PATH="$PATH" \
    TERM="${TERM:-xterm-256color}" \
    LANG="${LANG:-en_US.UTF-8}" \
    SSH_AUTH_SOCK="${SSH_AUTH_SOCK:-}" \
    SSH_AGENT_PID="${SSH_AGENT_PID:-}" \
    INPUT_DASHBOARD_PASS="${INPUT_DASHBOARD_PASS:-}" \
    INPUT_APP_NAME="${INPUT_APP_NAME:-}" \
    INPUT_DASHBOARD_URL="${INPUT_DASHBOARD_URL:-}" \
    INPUT_DASHBOARD_USER="${INPUT_DASHBOARD_USER:-}" \
    INPUT_CERTBOT_EMAIL="${INPUT_CERTBOT_EMAIL:-}" \
    INPUT_CERTBOT_STAGING="${INPUT_CERTBOT_STAGING:-}" \
    INPUT_CERTBOT_AUTO_RENEW="${INPUT_CERTBOT_AUTO_RENEW:-}" \
    INPUT_TAG="${INPUT_TAG:-}" \
    INPUT_REGISTRY_HOST="${INPUT_REGISTRY_HOST:-}" \
    INPUT_IMAGE_REPOSITORY="${INPUT_IMAGE_REPOSITORY:-}" \
    INPUT_REGISTRY_USERNAME="${INPUT_REGISTRY_USERNAME:-}" \
    INPUT_REGISTRY_PASSWORD="${INPUT_REGISTRY_PASSWORD:-}" \
    "$PAAS_BIN" run "$extension"
}

load_dashboard_defaults() {
  export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
  export INPUT_APP_NAME="paas-dashboard"
  export INPUT_DASHBOARD_URL="http://127.0.0.1:7000"
  export INPUT_DASHBOARD_USER="<DASHBOARD_USER>"
  export INPUT_CERTBOT_EMAIL="<CERTBOT_EMAIL>"
  export INPUT_CERTBOT_STAGING="false"
  export INPUT_CERTBOT_AUTO_RENEW="true"
  export INPUT_REGISTRY_HOST="<REGISTRY_HOST>"
  export INPUT_IMAGE_REPOSITORY="<REGISTRY_NAMESPACE>/<REPOSITORY>"
  export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
  export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
}

dashboard_bootstrap() {
  load_dashboard_defaults
  run_paas bootstrap-direct
}

dashboard_deploy_direct() {
  load_dashboard_defaults
  run_paas deploy-direct
}

dashboard_deploy() {
  load_dashboard_defaults
  run_paas deploy
}

usage() {
  cat <<'EOF'
Usage:
  ./.paas/paas-profiles.sh <command>

Commands:
  dashboard-bootstrap      first install / reinstall of the internal dashboard
  dashboard-deploy-direct  update dashboard with direct server build
  dashboard-deploy         update dashboard with registry-backed image

Run:
  bash ./.paas/paas-profiles.sh dashboard-deploy-direct
EOF
}

case "${1:-}" in
  dashboard-bootstrap) dashboard_bootstrap ;;
  dashboard-deploy-direct) dashboard_deploy_direct ;;
  dashboard-deploy) dashboard_deploy ;;
  *) usage ;;
esac
