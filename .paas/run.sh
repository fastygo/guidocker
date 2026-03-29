#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG_FILE="${ROOT_DIR}/.paas/config.yml"
PAAS_BIN="${PAAS_BIN:-${ROOT_DIR}/paas.exe}"

normalize_config_value() {
  local value="$1"
  if [[ "${value}" == '""' ]]; then
    printf ''
  elif [[ "${value}" == \"*\" && "${value}" == *\" ]]; then
    value="${value#\"}"
    value="${value%\"}"
    printf '%s' "${value}"
  else
    printf '%s' "${value}"
  fi
}

input_name_for_key() {
  printf '%s' "${1#INPUT_}" | tr 'A-Z' 'a-z'
}

mask_value() {
  local key="$1"
  local value="$2"
  case "${key}" in
    *PASS|*PASSWORD|*TOKEN|*SECRET)
      if [[ -n "${value}" ]]; then
        printf '********'
      else
        printf '""'
      fi
      ;;
    *)
      if [[ -n "${value}" ]]; then
        printf '%s' "${value}"
      else
        printf '""'
      fi
      ;;
  esac
}

usage() {
  cat <<'EOF'
Usage:
  bash ./.paas/run.sh [--yes] <extension> [additional paas args]

Behavior:
  1. Reads INPUT_* defaults from ./.paas/config.yml
  2. Applies exported INPUT_* environment variables as overrides
  3. Shows resolved inputs and asks for confirmation by default
  4. Calls paas.exe run <extension> with explicit --input key=value flags

Example:
  bash ./.paas/run.sh bootstrap-direct
  bash ./.paas/run.sh --yes bootstrap-direct
  INPUT_TAG=sha-override bash ./.paas/run.sh deploy-direct
  PAAS_ASSUME_YES=true bash ./.paas/run.sh deploy-direct
  bash ./.paas/run.sh bootstrap-direct --dry-run
EOF
}

ASSUME_YES="${PAAS_ASSUME_YES:-false}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes)
      ASSUME_YES="true"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    --)
      shift
      break
      ;;
    -*)
      echo "Unknown runner option: $1" >&2
      usage
      exit 1
      ;;
    *)
      break
      ;;
  esac
done

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

if [[ ! -f "${CONFIG_FILE}" ]]; then
  echo "Config file not found: ${CONFIG_FILE}" >&2
  exit 1
fi

EXTENSIONS_DIR_REL="$(awk -F': *' '/^extensions_dir:/ {print $2; exit}' "${CONFIG_FILE}")"
EXTENSIONS_DIR_REL="${EXTENSIONS_DIR_REL:-.paas/extensions}"
if [[ "${EXTENSIONS_DIR_REL}" = /* || "${EXTENSIONS_DIR_REL}" =~ ^[A-Za-z]:/ ]]; then
  EXTENSIONS_DIR="${EXTENSIONS_DIR_REL}"
else
  EXTENSIONS_DIR="${ROOT_DIR}/${EXTENSIONS_DIR_REL}"
fi

if [[ ! -x "${PAAS_BIN}" ]]; then
  echo "paas binary not found or not executable: ${PAAS_BIN}" >&2
  exit 1
fi

EXTENSION="$1"
shift

declare -A default_values=()
declare -A all_keys=()
declare -A extension_defaults=()
declare -A extension_has_default=()
declare -A extension_required=()
declare -A extension_inputs=()
declare -A resolved_values=()
declare -A value_sources=()
declare -A summary_keys=()
declare -a extension_key_order=()
declare -a warnings=()
declare -a notes=()

while IFS=$'\t' read -r key raw_value; do
  [[ -z "${key}" ]] && continue
  value="$(normalize_config_value "${raw_value}")"
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

EXTENSION_FILE="${EXTENSIONS_DIR}/${EXTENSION}.yml"
if [[ -f "${EXTENSION_FILE}" ]]; then
  current_name=""
  current_default=""
  current_has_default="false"
  current_required="false"
  in_inputs=0

  finalize_extension_input() {
    if [[ -z "${current_name}" ]]; then
      return
    fi

    local key
    key="INPUT_$(printf '%s' "${current_name}" | tr '[:lower:]' '[:upper:]')"
    extension_inputs["${key}"]=1
    extension_required["${key}"]="${current_required}"
    if [[ "${current_has_default}" == "true" ]]; then
      extension_defaults["${key}"]="${current_default}"
      extension_has_default["${key}"]="true"
    fi
    summary_keys["${key}"]=1
    extension_key_order+=("${key}")
  }

  while IFS= read -r raw_line || [[ -n "${raw_line}" ]]; do
    line="${raw_line%$'\r'}"
    if [[ "${line}" =~ ^inputs:[[:space:]]*$ ]]; then
      in_inputs=1
      continue
    fi
    if [[ "${in_inputs}" -eq 1 && "${line}" =~ ^steps:[[:space:]]*$ ]]; then
      finalize_extension_input
      break
    fi
    if [[ "${in_inputs}" -ne 1 ]]; then
      continue
    fi

    if [[ "${line}" =~ ^[[:space:]]*-[[:space:]]name:[[:space:]]*(.+)$ ]]; then
      finalize_extension_input
      current_name="${BASH_REMATCH[1]}"
      current_default=""
      current_has_default="false"
      current_required="false"
      continue
    fi
    if [[ -z "${current_name}" ]]; then
      continue
    fi
    if [[ "${line}" =~ ^[[:space:]]+required:[[:space:]]*true[[:space:]]*$ ]]; then
      current_required="true"
      continue
    fi
    if [[ "${line}" =~ ^[[:space:]]+default:[[:space:]]*(.*)$ ]]; then
      current_default="$(normalize_config_value "${BASH_REMATCH[1]}")"
      current_has_default="true"
    fi
  done < "${EXTENSION_FILE}"

  finalize_extension_input
fi

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

  input_name="$(input_name_for_key "${key}")"
  resolved_values["${key}"]="${value}"
  if [[ "${!key+x}" == x ]]; then
    value_sources["${key}"]="env"
  else
    value_sources["${key}"]="config"
  fi
  summary_keys["${key}"]=1
  args+=("--input" "${input_name}=${value}")
done < <(printf '%s\n' "${!all_keys[@]}" | sort)

while IFS= read -r key; do
  [[ -z "${key}" ]] && continue
  if [[ -v resolved_values["${key}"] ]]; then
    continue
  fi
  if [[ "${extension_has_default["${key}"]:-false}" == "true" ]]; then
    resolved_values["${key}"]="${extension_defaults["${key}"]}"
    value_sources["${key}"]="extension default"
    continue
  fi
  if [[ "${extension_required["${key}"]:-false}" == "true" ]]; then
    value_sources["${key}"]="missing"
    warnings+=("${key} is required by ${EXTENSION} but no value is currently resolved")
  fi
done < <(printf '%s\n' "${!summary_keys[@]}" | sort)

resolved_value_or_empty() {
  local key="$1"
  if [[ -v resolved_values["${key}"] ]]; then
    printf '%s' "${resolved_values["${key}"]}"
  else
    printf ''
  fi
}

case "${EXTENSION}" in
  app-bootstrap-direct|app-deploy-direct)
    public_domain_value="$(resolved_value_or_empty "INPUT_PUBLIC_DOMAIN")"
    use_tls_value="$(resolved_value_or_empty "INPUT_USE_TLS")"
    certbot_email_value="$(resolved_value_or_empty "INPUT_CERTBOT_EMAIL")"

    if [[ -z "${public_domain_value}" ]]; then
      notes+=("INPUT_PUBLIC_DOMAIN resolves empty; the app will not receive dashboard-managed domain routing until this is configured.")
    fi
    case "${use_tls_value}" in
      true|TRUE|True|1)
        if [[ -z "${public_domain_value}" ]]; then
          warnings+=("INPUT_USE_TLS resolves true, but INPUT_PUBLIC_DOMAIN is empty.")
        fi
        ;;
      *)
        if [[ -n "${public_domain_value}" ]]; then
          notes+=("INPUT_USE_TLS resolves false; dashboard-managed HTTPS routing will stay disabled for ${public_domain_value}.")
        fi
        ;;
    esac

    if [[ "${EXTENSION}" == "app-bootstrap-direct" ]]; then
      if [[ -z "${certbot_email_value}" ]]; then
        notes+=("INPUT_CERTBOT_EMAIL resolves empty, so bootstrap will skip platform settings seed via /api/settings.")
      else
        notes+=("Bootstrap can seed platform settings because INPUT_CERTBOT_EMAIL is resolved.")
      fi
    fi

    if [[ "${EXTENSION}" == "app-deploy-direct" ]]; then
      notes+=("app-deploy-direct re-applies app routing only and does not sync platform settings via /api/settings.")
    fi
    ;;
esac

if [[ $# -gt 0 ]]; then
  args+=("$@")
fi

echo "Extension: ${EXTENSION}"
if [[ -f "${EXTENSION_FILE}" ]]; then
  echo "Extension file: ${EXTENSION_FILE}"
fi
if [[ $# -gt 0 ]]; then
  echo "Additional paas args: $*"
fi
echo
while IFS= read -r key; do
  [[ -z "${key}" ]] && continue
  source="${value_sources["${key}"]:-missing}"
  display_value="<missing>"
  if [[ -v resolved_values["${key}"] ]]; then
    display_value="$(mask_value "${key}" "${resolved_values["${key}"]}")"
    if [[ "${display_value}" == '""' && "${extension_required["${key}"]:-false}" == "true" ]]; then
      warnings+=("${key} is required by ${EXTENSION} but resolves to an empty value")
      display_value='"" (empty)'
    fi
  elif [[ "${extension_required["${key}"]:-false}" == "true" ]]; then
    display_value="<missing required>"
  fi
  echo "Input:       ${key}"
  echo "Mapped name: $(input_name_for_key "${key}")"
  echo "Source:      ${source}"
  echo "Value:       ${display_value}"
  echo
done < <(printf '%s\n' "${!summary_keys[@]}" | sort)

if [[ "${#warnings[@]}" -gt 0 ]]; then
  echo "Warnings:"
  for warning in "${warnings[@]}"; do
    printf '  - %s\n' "${warning}"
  done
  echo
fi

if [[ "${#notes[@]}" -gt 0 ]]; then
  echo "Notes:"
  for note in "${notes[@]}"; do
    printf '  - %s\n' "${note}"
  done
  echo
fi

case "${ASSUME_YES}" in
  true|TRUE|True|1|yes|YES|Yes)
    echo "Confirmation skipped (--yes / PAAS_ASSUME_YES)."
    ;;
  *)
    read -r -p "Continue deployment? [y/N] " confirm
    case "${confirm}" in
      y|Y|yes|YES)
        ;;
      *)
        echo "Deployment aborted."
        exit 0
        ;;
    esac
    ;;
esac

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
