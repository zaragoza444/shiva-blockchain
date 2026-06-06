# OneX Token Lab

OneX-branded token generator and launcher — create **BEP-20 tokens on BNB Smart Chain mainnet**, auto-generate names/symbols, and view them on [BSCScan](https://bscscan.com) with live DEX prices.

## Features

- Real on-chain ERC-20 deploy (MetaMask or platform wallet)
- BSCScan / Etherscan V2 + on-chain RPC fallback
- DexScreener USD price, liquidity, 24h change
- **PancakeSwap V2 liquidity** — create TOKEN/BNB or TOKEN/USDT pools via MetaMask
- Production: API key auth, CORS, rate limits, health checks, Docker, nginx TLS

## Quick start (development)

```bat
run-onex-token-lab.bat
```

Open http://127.0.0.1:9340

## Production deploy

See **[DEPLOY.md](DEPLOY.md)** for full guide.

```powershell
.\scripts\deploy-bsc-launcher.ps1
```

Or:

```bash
cp bsc-launcher/.env.production.example bsc-launcher/.env
# edit keys and CORS
docker compose -f docker-compose.bsc-launcher.yml up -d --build
```

With HTTPS:

```bash
docker compose -f docker-compose.bsc-launcher.yml --profile proxy up -d
```

## Environment

| Variable | Required (prod) | Purpose |
|----------|-----------------|---------|
| `BSC_LAUNCHER_ENV` | yes | `production` |
| `BSC_LAUNCHER_API_KEY` | yes | Protects POST /api/deploy, /register |
| `BSC_LAUNCHER_CORS_ORIGINS` | yes | `https://yourdomain.com` |
| `BSCSCAN_API_KEY` | recommended | Holder/transfer stats |
| `BSC_RPC_URL` | optional | Default: public dataseed |
| `BSC_DEPLOYER_PRIVATE_KEY` | optional | Platform wallet deploy |

Templates: `.env.example` (dev), `.env.production.example` (prod)

## API

| Method | Path | Auth |
|--------|------|------|
| GET | `/health`, `/ready` | no |
| GET | `/api/config`, `/api/tokens`, `/api/bscscan/:addr`, `/api/price/:addr` | no |
| POST | `/api/deploy`, `/api/tokens/register` | `X-API-Key` when configured |
| GET | `/api/liquidity/pair?token=&quote=` | no |
| POST | `/api/liquidity/register` | `X-API-Key` when configured |
| GET | `/api/liquidity` | no |

## Security

- Never commit `.env` or private keys
- Use MetaMask deploy for end users (non-custodial)
- Platform deployer wallet: minimal BNB, hot wallet only
- Users enter API key via **Settings** in the UI (localStorage)

## Contract

`contracts/src/SimpleERC20.sol` — compile with `node contracts/compile.js`

## Layout

```
bsc-launcher/
├── server/          Go API
├── web/             UI
├── abi/             Compiled bytecode
├── contracts/       Solidity source
├── DEPLOY.md        Production guide
└── Dockerfile
```
