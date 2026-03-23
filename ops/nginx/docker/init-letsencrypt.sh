#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${DOMAIN:-}" || -z "${LETSENCRYPT_EMAIL:-}" ]]; then
  echo "DOMAIN and LETSENCRYPT_EMAIL must be set in environment." >&2
  exit 1
fi

mkdir -p ./ops/nginx/docker/www ./ops/nginx/docker/certbot

# Start nginx once so certbot can complete HTTP-01 challenge.
docker compose up -d nginx

# Issue cert (includes www SAN).
docker compose run --rm certbot certonly --webroot -w /var/www/certbot \
  --email "$LETSENCRYPT_EMAIL" \
  -d "$DOMAIN" -d "www.$DOMAIN" \
  --agree-tos --no-eff-email --non-interactive

# Reload nginx with issued cert.
docker compose exec nginx nginx -s reload

echo "Certificate issued and nginx reloaded."
