#!/usr/bin/env bash
#
# Recursively scans dashboard/views for class attributes, extracts Tailwind-like
# class names, and reports those NOT present in dashboard/pkg/twsx/ui8kit.map.json.
# Output is deduplicated and sorted.
#
# Run from repository root:
#   bash scripts/tailwind-unmapped-classes.sh
#
# Optional: jq for JSON parsing (fallback uses grep/sed)
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VIEWS_DIR="${REPO_ROOT}/dashboard/views"
MAP_FILE="${REPO_ROOT}/dashboard/pkg/twsx/ui8kit.map.json"

if [[ ! -f "$MAP_FILE" ]]; then
  echo "Error: ui8kit.map.json not found at $MAP_FILE" >&2
  exit 1
fi

# Build set of known classes from ui8kit.map.json (keys only)
# Also add base utilities for variant-prefixed keys (e.g. md:flex -> flex)
declare -A KNOWN
if command -v jq &>/dev/null; then
  map_keys=$(jq -r 'keys[]' "$MAP_FILE")
else
  # Fallback: extract JSON keys with grep/sed (keys are "key":)
  map_keys=$(grep -oE '"[^"]+":' "$MAP_FILE" | sed 's/[":]//g')
fi
while IFS= read -r key; do
  [[ -z "$key" ]] && continue
  KNOWN["$key"]=1
  if [[ "$key" == *":"* ]]; then
    base="${key#*:}"
    KNOWN["$base"]=1
  fi
done <<< "$map_keys"

# Extract base utility from a class (strip variant prefixes)
# Tailwind variants: sm, md, lg, xl, 2xl, hover, focus, active, disabled, dark, etc.
strip_variant() {
  local c="$1"
  while [[ "$c" == *":"* ]]; do
    c="${c#*:}"
  done
  echo "$c"
}

# Gather all class-like tokens from views (recursive)
# Matches class="..." and class='...', extracts Tailwind-like tokens
# Exclude template artifacts, custom classes, and malformed tokens
BLOCKLIST='^(eq|if|end|class|statusClass|[0-9]+)$'
ALL_CLASSES=$(grep -rohE 'class=["'"'"'][^"'"'"']*["'"'"']' "$VIEWS_DIR" 2>/dev/null | \
  sed -E 's/^class=["'"'"']//;s/["'"'"']$//' | \
  tr -s ' \t\n' '\n' | \
  grep -oE '[a-zA-Z0-9_-]+(:[a-zA-Z0-9_-]+)*(-[a-zA-Z0-9_-]+)*' | \
  grep -vE "$BLOCKLIST" | \
  grep -v '_' | \
  grep -v -- '-$' | \
  grep -v '^dashboard-' | \
  grep -v '^$' || true)

# Count occurrences per class (keep all, not just unique)
declare -A COUNTS
while IFS= read -r c; do
  [[ -z "$c" ]] && continue
  if [[ -z "${KNOWN[$c]}" ]]; then
    base=$(strip_variant "$c")
    if [[ -z "${KNOWN[$base]}" ]]; then
      ((COUNTS[$c]++)) || COUNTS[$c]=1
    fi
  fi
done <<< "$ALL_CLASSES"

# Output
if [[ ${#COUNTS[@]} -eq 0 ]]; then
  echo "All Tailwind classes found in views are present in ui8kit.map.json."
  exit 0
fi

echo "Unmapped Tailwind classes (${#COUNTS[@]} unique):"
echo ""
printf '%s\n' "${!COUNTS[@]}" | sort | while read -r c; do
  printf '%4d  %s\n' "${COUNTS[$c]}" "$c"
done
