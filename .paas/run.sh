#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG_FILE="${ROOT_DIR}/.paas/config.yml"
PAAS_BIN="${PAAS_BIN:-${ROOT_DIR}/paas.exe}"

usage() {
  cat <<'EOF'
Usage:
  bash ./.paas/run.sh <extension> [additional paas args]

Behavior:
  1. Reads INPUT_* defaults from ./.paas/config.yml
  2. Applies exported INPUT_* environment variables as overrides
  3. Calls paas.exe run <extension> with explicit --input key=value flags

Example:
  bash ./.paas/run.sh bootstrap-direct
  INPUT_USE_TLS=true bash ./.paas/run.sh deploy-direct
  bash ./.paas/run.sh bootstrap-direct --dry-run
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

if [[ ! -f "${CONFIG_FILE}" ]]; then
  echo "Config file not found: ${CONFIG_FILE}" >&2
  exit 1
fi

if [[ ! -x "${PAAS_BIN}" ]]; then
  echo "paas binary not found or not executable: ${PAAS_BIN}" >&2
  exit 1
fi

EXTENSION="$1"
shift

declare -A default_values=()
declare -A all_keys=()

while IFS=$'\t' read -r key raw_value; do
  [[ -z "${key}" ]] && continue
  value="${raw_value}"
  if [[ "${value}" == '""' ]]; then
    value=""
  elif [[ "${value}" == \"*\" && "${value}" == *\" ]]; then
    value="${value#\"}"
    value="${value%\"}"
  fi
  default_values["${key}"]="${value}"
  all_keys["${key}"]=1
done < <(
  awk '
    /^defaults:[[:space:]]*$/ { in_defaults=1; next }
    in_defaults && /^[^[:space:]]/ { in_defaults=0 }
    in_defaults && /^  [A-Za-z0-9_]+:/ {
      line=$0
      sub(/^  /, "", line)
      key=line
      sub(/:.*/, "", key)
      value=line
      sub(/^[^:]+:[[:space:]]*/, "", value)
      print key "\t" value
    }
  ' "${CONFIG_FILE}"
)

while IFS='=' read -r key _; do
  [[ "${key}" == INPUT_* ]] || continue
  all_keys["${key}"]=1
done < <(env)

args=("${PAAS_BIN}" "run" "${EXTENSION}")

while IFS= read -r key; do
  value=""
  if [[ "${!key+x}" == x ]]; then
    value="${!key}"
  elif [[ -v default_values["${key}"] ]]; then
    value="${default_values["${key}"]}"
  else
    continue
  fi

  input_name="$(printf '%s' "${key#INPUT_}" | tr 'A-Z' 'a-z')"
  args+=("--input" "${input_name}=${value}")
done < <(printf '%s\n' "${!all_keys[@]}" | sort)

if [[ $# -gt 0 ]]; then
  args+=("$@")
fi

env -i \
  HOME="${HOME}" \
  USERPROFILE="${USERPROFILE:-${HOME}}" \
  HOMEDRIVE="${HOMEDRIVE:-C:}" \
  HOMEPATH="${HOMEPATH:-/}" \
  PATH="${PATH}" \
  TERM="${TERM:-xterm-256color}" \
  LANG="${LANG:-en_US.UTF-8}" \
  SSH_AUTH_SOCK="${SSH_AUTH_SOCK:-}" \
  SSH_AGENT_PID="${SSH_AGENT_PID:-}" \
  "${args[@]}"
