# OneX Wallet — production go-live

## One command (GitHub + Gitea)

```powershell
# 1. Copy and edit remotes
copy remotes.env.example remotes.env
# Set GITEA_URL and optional ONEX_BRIDGE_PUBLIC_URL

# 2. Publish
.\scripts\publish-production.ps1
```

With bridge URL (Render):

```powershell
.\scripts\publish-production.ps1 -BridgeUrl "https://your-onex-bridge.onrender.com"
```

## What gets published

| Target | Result |
|--------|--------|
| **GitHub** | Repo `zaragoza444/onex-blockchain`, Actions Pages workflow |
| **Wallet UI** | https://zaragoza444.github.io/onex-blockchain/wallet/ |
| **Gitea** | Push to `GITEA_URL`, `.gitea/workflows/pages.yml` for Pages |
| **Bridge API** | Deploy [`render.yaml`](../render.yaml) on Render (or Docker prod) |

## After first push

### GitHub Pages
1. Repo → **Settings → Pages** → source **GitHub Actions**
2. **Actions → GitHub Pages** → confirm green run
3. Optional: **Settings → Actions → Variables** → `ONEX_BRIDGE_PUBLIC_URL`

### Gitea Pages
1. Push to your Gitea remote (`remotes.env` → `GITEA_URL`)
2. Repo → **Settings → Pages** → enable Actions deploy
3. Set variable `ONEX_BRIDGE_PUBLIC_URL` if using split hosting

### Bridge (required for send/swap/wallet sync)
1. [Render Blueprints](https://dashboard.render.com/blueprints) → connect repo → apply `render.yaml`
2. Copy **onex-bridge** HTTPS URL
3. Run: `.\scripts\connect-bridge.ps1 -BridgeUrl "https://..." -GitHubVariable`

### Mobile app
Default wallet URL: `https://zaragoza444.github.io/onex-blockchain/wallet/` (`mobile/.env.example`)

## CORS

Set on bridge (`ONEX_CORS_ORIGINS`):

```env
ONEX_CORS_ORIGINS=https://zaragoza444.github.io,https://your-gitea-pages-host
```

## Local production stack

```powershell
docker compose -f docker-compose.prod.yml up -d --build
# Wallet: http://HOST:9338/wallet/
```

See [DEPLOY.md](../DEPLOY.md) and [docs/HOSTING.md](HOSTING.md).
