# Deploy OneX to novatrustee.digital

> **Note:** DNS is registered as **novatrustee.digital** (not `.digitall`). Point all A records at your **OneX VPS** IP (remove parking IPs if the host does not serve the app yet).

## 1. DNS

At your registrar for **novatrustee.digital**:

| Type | Host / name | Value |
|------|-------------|--------|
| A | `novatrustee` or `@` | **One** VPS public IPv4 (remove extra/parking A records) |

Current DNS often shows multiple parking IPs until you point the name at your server.

Wait until it resolves:

```bash
dig +short novatrustee.digital
```

**Windows preflight:** `.\scripts\deploy-vps-novatrustee.ps1 -VpsIp YOUR_VPS_IP`

### Parking / wrong DNS (site times out)

If preflight shows these IPs, the name is **not** on your server yet:

`107.161.23.204`, `198.251.81.30`, `209.141.38.71`

At your registrar: **delete** all A records for `novatrustee.digital`, add **one** A record → your VPS IPv4, wait 5–30 minutes, re-run preflight until only your IP appears.

## 2. Server (Ubuntu VPS)

```bash
sudo apt update && sudo apt install -y docker.io docker-compose-plugin certbot
sudo usermod -aG docker $USER
# log out and back in

git clone <your-repo-url> onex-blockchain
cd onex-blockchain
cp deploy/env.novatrustee.digital.example .env
nano .env   # set ONEX_API_KEY
```

## 3. TLS (Let's Encrypt)

Stop anything on ports 80/443, then:

```bash
sudo certbot certonly --standalone -d novatrustee.digital \
  --agree-tos -m admin@digital --non-interactive
mkdir -p deploy/certs
sudo cp /etc/letsencrypt/live/novatrustee.digital/fullchain.pem deploy/certs/
sudo cp /etc/letsencrypt/live/novatrustee.digital/privkey.pem deploy/certs/
sudo chmod 600 deploy/certs/privkey.pem
```

Or run: `chmod +x scripts/deploy-vps-novatrustee.sh && CERTBOT_EMAIL=you@digital ./scripts/deploy-vps-novatrustee.sh`

The script stops any running stack, runs certbot, copies certs, and starts Docker with the TLS proxy.

## 4. Start stack

```bash
docker compose -f docker-compose.prod.yml --profile proxy up -d --build
docker compose -f docker-compose.prod.yml ps
```

## 5. URLs

| Service | URL |
|---------|-----|
| Wallet | https://novatrustee.digital/wallet/ |
| Bridge status | https://novatrustee.digital/bridge/platform/status |
| Explorer | https://novatrustee.digital/explorer/ |

Wallet on this host uses same-origin bridge calls (no API key in Settings).

Gitea Pages wallet: Settings → bridge `https://novatrustee.digital` + API key from `.env`.

## 6. Verify

```bash
curl -s https://novatrustee.digital/bridge/platform/status
curl -s -o /dev/null -w "%{http_code}\n" https://novatrustee.digital/wallet/
```

## 7. Renew certs (cron)

```bash
sudo certbot renew --deploy-hook "cd /path/to/onex-blockchain && sudo cp /etc/letsencrypt/live/novatrustee.digital/*.pem deploy/certs/ && docker compose -f docker-compose.prod.yml --profile proxy restart nginx"
```
