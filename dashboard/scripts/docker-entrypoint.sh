#!/usr/bin/env bash
set -euo pipefail

NGINX_BINARY="${PAAS_NGINX_BINARY:-nginx}"
SITES_DIR="${PAAS_NGINX_SITES_DIR:-/etc/nginx/conf.d}"

if ! command -v "$NGINX_BINARY" >/dev/null 2>&1; then
	echo "docker-entrypoint: nginx binary not found: $NGINX_BINARY" >&2
	exit 1
fi

mkdir -p /run/nginx "$SITES_DIR"
mkdir -p /etc/letsencrypt /var/lib/letsencrypt /var/log/letsencrypt

if ! "$NGINX_BINARY" -t >/dev/null 2>&1; then
	echo "docker-entrypoint: nginx configuration test failed, check generated app routes" >&2
	exit 1
fi

"$NGINX_BINARY" -g "daemon off;" &
NGINX_PID=$!

cleanup() {
	if kill -0 "$NGINX_PID" >/dev/null 2>&1; then
		kill "$NGINX_PID" >/dev/null 2>&1 || true
	fi
	if kill -0 "$DASHBOARD_PID" >/dev/null 2>&1; then
		kill "$DASHBOARD_PID" >/dev/null 2>&1 || true
	fi
}
trap cleanup EXIT TERM INT

/usr/local/bin/dashboard "$@" &
DASHBOARD_PID=$!

wait -n "$NGINX_PID" "$DASHBOARD_PID"
EXIT_CODE=$?
cleanup
exit "$EXIT_CODE"

