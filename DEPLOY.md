# Production deployment

Run OneX blockchain + OKX-style wallet bridge on a server (Docker or systemd).

## Docker (recommended)

```bash
git clone <your-repo-url> onex-blockchain
cd onex-blockchain
cp .env.example .env
# Edit .env: ONEX_API_KEY, ONEX_CORS_ORIGINS=https://your-domain.com

docker compose -f docker-compose.prod.yml up -d --build
```

| Service | Port | URL |
|---------|------|-----|
| Node API + explorer | 8545 | `http://HOST:8545/explorer/` |
| Wallet (bridge) | 9338 | `http://HOST:9338/wallet/` |

With TLS reverse proxy:

```bash
mkdir -p deploy/certs
# Place fullchain.pem + privkey.pem in deploy/certs/
docker compose -f docker-compose.prod.yml --profile proxy up -d
```

- Wallet: `https://your-domain/wallet/`
- Explorer: `https://your-domain/explorer/`

## systemd (bare metal)

```bash
sudo mkdir -p /opt/onex/bin /var/lib/onex /var/lib/onex-bridge/wallets
sudo cp bin/onexd bin/onex bin/onex-bridge /opt/onex/bin/
sudo cp -r configs /opt/onex/
sudo cp deploy/onexd.service deploy/onex-bridge.service /etc/systemd/system/
sudo cp .env.example /etc/onex/onex.env
# Edit /etc/onex/onex.env

sudo systemctl daemon-reload
sudo systemctl enable --now onexd onex-bridge
```

## Security checklist

- Set `ONEX_API_KEY` on the node for `POST /api/v1/tx` and `/rpc`
- Restrict `ONEX_CORS_ORIGINS` to your wallet domain
- Use TLS (`--profile proxy` or nginx in front)
- Never commit `.env`, wallet JSON, or private keys
- Testnet faucet: set `ONEX_FAUCET_PRIVATE_KEY` only on testnet hosts

## OneX Wallet mobile apps

React Native (Expo) WebView app in [`mobile/`](mobile/).

1. Deploy backend with HTTPS (`https://YOUR_DOMAIN/wallet/`).
2. Set `EXPO_PUBLIC_WALLET_URL` in `mobile/.env` (see `mobile/.env.example`).
3. Build and publish: **[mobile/PUBLISH.md](mobile/PUBLISH.md)**.

```bash
cd mobile && npm install && eas build --platform all --profile production
```

## Publish to GitHub + Gitea

```powershell
# Windows
.\scripts\publish-remotes.ps1 -GitHub "git@github.com:USER/onex-blockchain.git" -Gitea "git@git.example.com:USER/onex-blockchain.git"
```

```bash
# Linux
./scripts/publish-remotes.sh git@github.com:USER/onex-blockchain.git git@git.example.com:USER/onex-blockchain.git
```

Create empty repos on GitHub and Gitea first, then run the script.
