#!/usr/bin/env bash
set -euo pipefail

# Reload nginx only when certs are renewed.
if command -v systemctl >/dev/null 2>&1; then
  sudo systemctl reload nginx
elif command -v brew >/dev/null 2>&1 && command -v nginx >/dev/null 2>&1; then
  nginx -s reload
else
  echo "No supported nginx reload method found" >&2
  exit 1
fi
