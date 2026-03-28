#!/usr/bin/env bash

# Generic presets for deployment without editing .paas/config.yml for every run.
# Copy app values and run one of the command functions.

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
    INPUT_APP_ID="${INPUT_APP_ID:-}" \
    INPUT_DASHBOARD_URL="${INPUT_DASHBOARD_URL:-}" \
    INPUT_DASHBOARD_USER="${INPUT_DASHBOARD_USER:-}" \
    INPUT_PUBLIC_DOMAIN="${INPUT_PUBLIC_DOMAIN:-}" \
    INPUT_PROXY_TARGET_PORT="${INPUT_PROXY_TARGET_PORT:-}" \
    INPUT_USE_TLS="${INPUT_USE_TLS:-}" \
    INPUT_TAG="${INPUT_TAG:-}" \
    INPUT_HEALTHCHECK_URL="${INPUT_HEALTHCHECK_URL:-}" \
    INPUT_REGISTRY_HOST="${INPUT_REGISTRY_HOST:-}" \
    INPUT_IMAGE_REPOSITORY="${INPUT_IMAGE_REPOSITORY:-}" \
    INPUT_REGISTRY_USERNAME="${INPUT_REGISTRY_USERNAME:-}" \
    INPUT_REGISTRY_PASSWORD="${INPUT_REGISTRY_PASSWORD:-}" \
    "$PAAS_BIN" run "$extension"
}

load_appfasty_defaults() {
  # Replace placeholders before use.
  export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
  export INPUT_APP_NAME="appfasty"
  export INPUT_DASHBOARD_URL="http://<DASHBOARD_HOST>:<DASHBOARD_PORT>"
  export INPUT_DASHBOARD_USER="<DASHBOARD_USER>"
  export INPUT_PUBLIC_DOMAIN="<APP_PUBLIC_DOMAIN>"
  export INPUT_PROXY_TARGET_PORT="80"
  export INPUT_USE_TLS="false"
  export INPUT_HEALTHCHECK_URL="https://<APP_PUBLIC_DOMAIN>/api/health"
  export INPUT_APP_ID="<APP_ID>"
  export INPUT_REGISTRY_HOST="<REGISTRY_HOST>"
  export INPUT_IMAGE_REPOSITORY="<REGISTRY_NAMESPACE>/<REPOSITORY>"
  export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
  export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
}

load_demo_defaults() {
  # Replace placeholders before use.
  export INPUT_DASHBOARD_PASS="<DASHBOARD_PASSWORD>"
  export INPUT_APP_NAME="demo-app"
  export INPUT_DASHBOARD_URL="http://<DASHBOARD_HOST>:<DASHBOARD_PORT>"
  export INPUT_DASHBOARD_USER="<DASHBOARD_USER>"
  export INPUT_PUBLIC_DOMAIN="<DEMO_APP_PUBLIC_DOMAIN>"
  export INPUT_PROXY_TARGET_PORT="80"
  export INPUT_USE_TLS="false"
  export INPUT_HEALTHCHECK_URL=""
  export INPUT_APP_ID="<DEMO_APP_ID>"
  export INPUT_REGISTRY_HOST="<REGISTRY_HOST>"
  export INPUT_IMAGE_REPOSITORY="<DEMO_REGISTRY_NAMESPACE>/<DEMO_REPOSITORY>"
  export INPUT_REGISTRY_USERNAME="<REGISTRY_USERNAME>"
  export INPUT_REGISTRY_PASSWORD="<REGISTRY_PASSWORD>"
}

appfasty_bootstrap() {
  load_appfasty_defaults
  export INPUT_APP_ID=""
  run_paas bootstrap-direct
}

appfasty_deploy_direct() {
  load_appfasty_defaults
  run_paas deploy-direct
}

appfasty_deploy() {
  load_appfasty_defaults
  run_paas deploy
}

demo_bootstrap() {
  load_demo_defaults
  export INPUT_APP_ID=""
  run_paas bootstrap-direct
}

demo_deploy_direct() {
  load_demo_defaults
  run_paas deploy-direct
}

demo_deploy() {
  load_demo_defaults
  run_paas deploy
}

usage() {
  cat <<'EOF'
Usage:
  ./.paas/paas-profiles.sh <command>

Commands:
  appfasty-bootstrap      bootstrap-first deploy for appfasty profile
  appfasty-deploy-direct  update existing app (direct image on server)
  appfasty-deploy         deploy through registry flow
  demo-bootstrap          bootstrap-first deploy for demo profile
  demo-deploy-direct      update existing app (direct image on server)
  demo-deploy             deploy through registry flow

Run:
  bash ./.paas/paas-profiles.sh appfasty-deploy-direct
EOF
}

case "${1:-}" in
  appfasty-bootstrap) appfasty_bootstrap ;;
  appfasty-deploy-direct) appfasty_deploy_direct ;;
  appfasty-deploy) appfasty_deploy ;;
  demo-bootstrap) demo_bootstrap ;;
  demo-deploy-direct) demo_deploy_direct ;;
  demo-deploy) demo_deploy ;;
  *) usage ;;
esac
