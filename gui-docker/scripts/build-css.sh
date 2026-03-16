#!/usr/bin/env bash
set -euo pipefail
npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/app.css
