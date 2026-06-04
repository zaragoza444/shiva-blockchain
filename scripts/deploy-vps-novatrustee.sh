#!/usr/bin/env bash
# Production deploy for novatrustee.digital (run on Ubuntu VPS with Docker).
set -euo pipefail

DOMAIN="${ONEX_DEPLOY_DOMAIN:-novatrustee.digital}"
EMAIL="${CERTBOT_EMAIL:-admin@digital}"
COMPOSE="docker compose -f docker-compose.prod.yml"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> DNS check: $DOMAIN"
if command -v dig >/dev/null 2>&1; then
  mapfile -t IPS < <(dig +short "$DOMAIN" A | sort -u)
else
  mapfile -t IPS < <(getent ahosts "$DOMAIN" 2>/dev/null | awk '{print $1}' | sort -u)
fi
if [[ ${#IPS[@]} -eq 0 ]]; then
  echo "ERROR: No A record for $DOMAIN (use novatrustee.digital, not .digitall)."
  exit 1
fi
if [[ ${#IPS[@]} -gt 1 ]]; then
  echo "WARN: Multiple A records (${IPS[*]}). Use ONE A record → this server's public IP."
fi
echo "    $DOMAIN -> ${IPS[0]}${IPS[1]:+, ${IPS[1]}...}"

if [[ ! -f .env ]]; then
  cp deploy/env.novatrustee.digital.example .env
  echo "Created .env — set ONEX_API_KEY, then re-run."
  exit 1
fi
sed -i 's/novatrustee\.digitall/novatrustee.digital/g' .env 2>/dev/null || \
  sed -i '' 's/novatrustee\.digitall/novatrustee.digital/g' .env 2>/dev/null || true
if grep -q 'CHANGE_ME' .env; then
  echo "ERROR: Set ONEX_API_KEY in .env before deploying."
  exit 1
fi

echo "==> Stop stack (free ports 80/443 for certbot)"
$COMPOSE --profile proxy down 2>/dev/null || true

echo "==> TLS certificate (Let's Encrypt standalone)"
sudo certbot certonly --standalone -d "$DOMAIN" --agree-tos -m "$EMAIL" --non-interactive

mkdir -p deploy/certs
sudo cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" deploy/certs/
sudo cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" deploy/certs/
sudo chmod 600 deploy/certs/privkey.pem

echo "==> Build and start (node + bridge + nginx)"
$COMPOSE --profile proxy up -d --build

echo "==> Wait for health"
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -sf "http://127.0.0.1/health" >/dev/null 2>&1 || \
     curl -sfk "https://127.0.0.1/health" >/dev/null 2>&1; then
    break
  fi
  sleep 3
done

if curl -sf "https://$DOMAIN/bridge/platform/status" | head -c 300; then
  echo ""
  echo "OK: https://$DOMAIN/wallet/"
else
  echo "WARN: External HTTPS check failed."
  echo "      Ensure DNS A record is ONLY this server and ports 80/443 are open."
  $COMPOSE --profile proxy ps
fi
