# Shiva Blockchain

Production proof-of-work blockchain: Ed25519 accounts, P2P sync, REST + JSON-RPC, embedded explorer, browser wallet (MetaMask-style UX), optional Chrome extension.

## Quick start (Windows)

```bat
build-shiva.bat
run-shiva.bat
```

- Explorer & wallet: http://localhost:8545/explorer/
- JSON-RPC: http://localhost:8545/rpc
- Health: http://localhost:8545/health

## Shiva Wallet — full DeFi UI

Open **http://127.0.0.1:9338/wallet/** for:

| Feature | Description |
|---------|-------------|
| **Portfolio** | Multi-chain & multi-token balances |
| **Send** | SHIVA on-chain + portfolio tokens |
| **Deposit** | Bridge in from 13+ chains |
| **Shiva Swap** | Uniswap-style AMM (x·y=k), liquidity pools, 0.3% fee |
| **Bridge tab** | Cross-chain via SHIVA hub pools |
| **NFT** | Mint & view Shiva NFTs |
| **Tasks** | Earn rewards (SHIVA, wSHIVA) |
| **Loans** | Borrow / lend with collateral |
| **Shiva AI** | Wallet assistant (local + optional cloud LLM) |

Chains: Shiva, Ethereum, BSC, Polygon, Arbitrum, Optimism, Avalanche, Base, Solana, Bitcoin, TRON, ALLTRA, testnet.

### Shiva AI

- **Wallet tab “AI”** — chat with portfolio-aware help
- **API:** `POST /bridge/ai/chat` (wallet) · `POST /api/v1/ai/chat` (node)
- **CLI:** `shiva-ai -ask "How do I swap?"` or interactive mode
- **Cloud mode:** set `SHIVA_AI_API_KEY` (OpenAI-compatible endpoint via `SHIVA_AI_BASE_URL`)
- **Local mode:** works without a key (built-in answers + navigation hints)

Configs: `configs/chains.json`, `tokens.json`, `swap-pairs.json`

## Shiva Wallet bridge (local)

The **bridge** (`shiva-bridge`) connects your wallet to the node on this PC:

| Component | URL |
|-----------|-----|
| Node | http://127.0.0.1:8545 |
| Bridge + Wallet UI | http://127.0.0.1:9338/wallet/ |
| Bridge JSON-RPC | http://127.0.0.1:9338/rpc |

```bat
run-shiva-wallet.bat
```

Config: `%USERPROFILE%\.shiva\bridge.json` · Wallet file: `%USERPROFILE%\.shiva\wallets\default.json`

Fix desktop shortcut: `powershell -File install-wallet-shortcut.ps1`

## Wallet (like MetaMask)

Shiva uses **Ed25519** (64-character hex addresses), not Ethereum. **Standard MetaMask cannot sign Shiva transactions.**

| Option | How |
|--------|-----|
| **Built-in wallet** | Open `/explorer/` → **Wallet** tab → Create / Import → Connect |
| **Browser extension** | Chrome → Load unpacked → `extension/` folder |
| **CLI** | `shiva wallet-create` · `shiva send -wallet ~/.shiva/wallets/default.json -to ADDR -amount 1` |
| **dApp API** | `window.shiva.request({ method: 'shiva_requestAccounts' })` |

### Add network to MetaMask (read-only chain info)

MetaMask can show the network but **will not sign** Shiva txs. Use Shiva Wallet for sends.

```javascript
// chainId 9001 = 0x2329 (mainnet)
await window.ethereum?.request({
  method: 'wallet_addEthereumChain',
  params: [{
    chainId: '0x2329',
    chainName: 'Shiva Mainnet',
    nativeCurrency: { name: 'Shiva', symbol: 'SHIVA', decimals: 8 },
    rpcUrls: ['https://your-node.example/rpc'],
    blockExplorerUrls: ['https://your-node.example/explorer/'],
  }],
});
```

## Production deployment

Full guide: **[DEPLOY.md](DEPLOY.md)**

### Docker (node + wallet bridge)

```bash
cp .env.example .env
docker compose -f docker-compose.prod.yml up -d --build
```

- Node: `http://HOST:8545` · Wallet: `http://HOST:9338/wallet/`
- TLS: `docker compose -f docker-compose.prod.yml --profile proxy up -d`

### Docker (node only)

```bash
docker compose up -d shivad
```

Testnet + faucet:

```bash
export SHIVA_FAUCET_PRIVATE_KEY=<hex>
docker compose --profile testnet up -d shivad-testnet
```

### TLS + API key

```bash
shivad -tls-cert /etc/shiva/tls/fullchain.pem -tls-key /etc/shiva/tls/privkey.pem -api-key "$SHIVA_API_KEY"
```

Environment (see `.env.example`):

- `SHIVA_FAUCET_PRIVATE_KEY` — testnet faucet
- `SHIVA_API_KEY` — protect `POST /api/v1/tx` and `/rpc`
- `SHIVA_CORS_ORIGINS` — browser wallet origins

### systemd + nginx

- `deploy/shivad.service` — node unit file
- `deploy/nginx.conf` — reverse proxy (use with `docker compose --profile proxy`)

## JSON-RPC methods

| Method | Description |
|--------|-------------|
| `shiva_chainId` | String chain id |
| `eth_chainId` | `0x2329` (network id 9001) |
| `shiva_getBalance` | Balance + nonce |
| `shiva_getTransactionCount` | Account nonce |
| `shiva_sendTransaction` | Submit signed tx |
| `eth_getBalance` / `eth_getTransactionCount` | Hex-encoded compat |

## Networks

| Network | Chain ID | Network ID | Genesis |
|---------|----------|------------|---------|
| Mainnet | shiva-mainnet-1 | 9001 | `configs/genesis.json` |
| Testnet | shiva-testnet-1 | 9002 | `configs/genesis-testnet.json` |

## Build

```bash
make build
make test
make run-testnet   # :8547, mining + faucet
```

## Publish to GitHub & Gitea

1. Create empty repos named `shiva-blockchain` on [GitHub](https://github.com/new) and your Gitea server.
2. Run (replace URLs with yours — see `remotes.example.env`):

```powershell
.\scripts\publish-remotes.ps1 `
  -GitHub "git@github.com:YOUR_USER/shiva-blockchain.git" `
  -Gitea "git@git.YOUR_DOMAIN:YOUR_USER/shiva-blockchain.git"
```

CI runs on push: `.github/workflows/ci.yml` (GitHub) and `.gitea/workflows/ci.yml` (Gitea).

## Layout

- `cmd/shivad` — node
- `cmd/shiva` — CLI wallet
- `cmd/shiva-bridge` — wallet bridge + OKX-style DeFi UI
- `internal/api` — REST, explorer, middleware
- `internal/bridge` — portfolio, swap, staking, static wallet
- `internal/rpc` — JSON-RPC
- `extension/` — Shiva Wallet (Chrome)
- `deploy/` — nginx, systemd
