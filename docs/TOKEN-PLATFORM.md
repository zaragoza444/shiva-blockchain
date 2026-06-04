# OneX Token Platform

Launch, manage, and wrap tokens across **13+ chains** from one interface — wallet UI, REST API, or CLI.

## Supported chains

| Chain | Type | Deploy standard |
|-------|------|-----------------|
| OneX Mainnet / Testnet | `onex` | Native registry |
| Ethereum, BSC, Polygon, Arbitrum, Optimism, Avalanche, Base, ALLTRA | `evm` | ERC-20 metadata |
| Solana | `solana` | SPL mint metadata |
| Bitcoin | `btc` | Asset reference |
| TRON | `tron` | TRC-20 metadata |

Config: `configs/chains.json`, `configs/token-platform.json`

## Quick start

1. Start node and bridge:

```bat
run-onex.bat
run-onex-wallet.bat
```

2. Open **Discover → Token Platform** at http://127.0.0.1:9338/wallet/

3. Or use CLI:

```bat
onex token-create -name "My Coin" -symbol MYC -supply 1000000 -chain ethereum
onex token-list
onex token-wrap -from-chain ethereum -from-token MYC -to-chain bsc -amount 500
onex platform-status
```

Set bridge URL if not local: `-bridge http://127.0.0.1:9338` or `ONEX_BRIDGE_URL`.

## Architecture

```
Wallet UI / CLI
       │
       ▼
  onex-bridge  ──►  Token Platform  ──►  Chain adapters (onex, evm, solana, …)
       │                    │
       │                    ├── platform-tokens.json  (~/.onex/)
       │                    └── platform-wraps.json
       ▼
  Portfolio + Registry + OneX Swap
```

### Deploy flow

1. Pick chain → adapter generates chain-specific contract/mint address and deploy payload
2. Token saved to `~/.onex/platform-tokens.json`
3. Registered in wallet registry + full supply credited to creator portfolio
4. Available for send, swap, stake, and cross-chain wrap

### Wrap flow

1. Lock origin token balance on source chain (portfolio debit)
2. Deploy or extend wrapped token on target chain (`w` + symbol)
3. Credit wrapped balance on target chain
4. Record stored in `~/.onex/platform-wraps.json`

## REST API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/bridge/platform/status` | Platform stats |
| GET | `/bridge/platform/tokens` | All deployed tokens |
| GET | `/bridge/platform/token?chain=ID&id=TOKEN` | Token detail |
| POST | `/bridge/platform/deploy` | Deploy token |
| POST | `/bridge/platform/wrap` | Cross-chain wrap |
| GET | `/bridge/platform/wraps` | Wrap history |

Legacy: `POST /bridge/tokens/create` still works (delegates to platform deploy).

### Deploy example

```json
POST /bridge/platform/deploy
{
  "chainId": "polygon",
  "name": "Polygon Coin",
  "symbol": "PCOIN",
  "decimals": 8,
  "supply": "100000000000"
}
```

Response includes `contractAddress`, `deployStatus`, `deployPayload` for external wallet signing on live networks.

### Wrap example

```json
POST /bridge/platform/wrap
{
  "originChainId": "ethereum",
  "originTokenId": "PCOIN",
  "targetChainId": "bsc",
  "amount": "50000000000"
}
```

## Storage

| File | Contents |
|------|----------|
| `~/.onex/platform-tokens.json` | Deployed tokens + contract metadata |
| `~/.onex/platform-wraps.json` | Cross-chain wrap records |
| `~/.onex/custom-tokens.json` | Registry mirror (backward compatible) |

## Live mainnet deploy

EVM and Solana adapters return **deploy payloads** (constructor args, RPC, explorer links). Use MetaMask, Phantom, or your signer with the payload to broadcast on live networks. OneX-native chains register instantly in the portfolio.

## Code layout

```
internal/bridge/chains/       # Per-chain deploy adapters
internal/bridge/tokenplatform/ # Store, config, types
internal/bridge/tokenplatform_bridge.go  # Bridge integration
internal/bridge/handlers_platform.go     # HTTP routes
cmd/onex/token.go             # CLI commands
configs/token-platform.json   # Platform config
```
