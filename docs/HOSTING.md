# GitHub Pages + bridge API

| Component | URL |
|-----------|-----|
| Wallet UI (Pages) | https://zaragoza444.github.io/shiva-blockchain/wallet/ |
| Bridge API | Your hosted `shiva-bridge` (required for send/swap/AI) |

**Quick connect (external):** open  
`https://zaragoza444.github.io/shiva-blockchain/wallet/?bridge=https://YOUR-bridge.onrender.com`  
or set the URL under **Settings → Bridge API**.

## Step 1 — Deploy bridge (Render, recommended)

1. Open [Render Dashboard](https://dashboard.render.com/) → **New** → **Blueprint**
2. Connect repo `zaragoza444/shiva-blockchain`
3. Apply [`render.yaml`](../render.yaml) (creates `shiva-node` + `shiva-bridge`)
4. When finished, copy the **shiva-bridge** public URL, e.g. `https://shiva-bridge-xxxx.onrender.com`

Free tier sleeps after inactivity; use a paid plan or your own VPS for production.

## Step 2 — Wire bridge URL into Pages wallet

**Option A — GitHub variable (best)**

1. Repo → **Settings** → **Secrets and variables** → **Actions** → **Variables**
2. Add variable: `SHIVA_BRIDGE_PUBLIC_URL` = `https://shiva-bridge-xxxx.onrender.com` (no trailing slash)
3. **Actions** → **GitHub Pages** → **Run workflow**

**Option B — Local script**

```powershell
.\scripts\set-bridge-url.ps1 -BridgeUrl "https://shiva-bridge-xxxx.onrender.com"
git add docs/wallet/config.js
git commit -m "Configure bridge URL for Pages wallet"
git push
```

## Step 3 — CORS on bridge

On Render, `SHIVA_CORS_ORIGINS` is set to `https://zaragoza444.github.io` in `render.yaml`.

If you host bridge elsewhere, set:

```env
SHIVA_CORS_ORIGINS=https://zaragoza444.github.io,https://your-bridge-host
```

## Step 4 — Mobile app

[`mobile/.env`](../mobile/.env) uses the Pages wallet URL. No change needed if you use GitHub Pages UI.

## Enable GitHub Pages

**Settings → Pages → Source: GitHub Actions**

Push to `main` runs [`.github/workflows/pages.yml`](../.github/workflows/pages.yml).

One command (after [GitHub CLI](https://cli.github.com/) is installed):

```powershell
.\scripts\push-and-enable-pages.ps1
# optional bridge URL for swaps on Pages:
.\scripts\push-and-enable-pages.ps1 -BridgeUrl "https://your-shiva-bridge.onrender.com"
```

## Full stack on your domain (alternative)

Use [DEPLOY.md](../DEPLOY.md) with `docker-compose.prod.yml` and set:

- `EXPO_PUBLIC_WALLET_URL=https://your-domain.com/wallet/`
- No split `SHIVA_BRIDGE_URL` needed (same origin)
