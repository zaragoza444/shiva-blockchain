# OneX Blockchain

Go monorepo for the OneX proof-of-work blockchain plus companion tools. Single Go module (`go.mod`, Go 1.22) builds every binary; frontends (explorer, wallet, token lab) are static assets embedded in / served by the Go binaries. Only `mobile/` (Expo) uses npm.

## Cursor Cloud specific instructions

Dependencies (`go mod download`, and `npm install` in `mobile/`) are refreshed by the startup update script — do not redo that here. Standard commands live in the `Makefile` and `README.md`; the notes below are the non-obvious caveats.

### Build / lint / test
- `make build` builds `onexd`, `onex`, `onex-bridge`, `onex-ai`. The BSC Token Launcher is a separate `main`: `go build -o bin/bsc-launcher ./bsc-launcher/server`.
- There is no Go linter configured. "Lint" = `go vet ./...` (+ `go build ./...`). CI (`.github/workflows/ci.yml`) only does build + `go test`.
- Tests: `make test` (`go test ./internal/...`). Most packages have no test files; only `internal/ai`, `internal/chain`, `internal/bridge/chains`, and `internal/legacy` do.

### Running the core stack (node + wallet) locally
- Docker is NOT required for development; run the built binaries directly. (`docker compose` files exist but Docker isn't installed in this env.)
- Run the testnet node via `make run-testnet`. It listens on API `:8547`, P2P `:30305` (mainnet `make run-node` uses `:8545` / `:30303`).
- The `p2p: bootstrap 127.0.0.1:30303 ... connection refused` log line is benign on a standalone local node (no peer to dial).
- The wallet **bridge** (`onex-bridge`) defaults its node URL to `:8545`. When pointing it at the testnet node you MUST pass `-node http://127.0.0.1:8547`, e.g. `./bin/onex-bridge -node http://127.0.0.1:8547 -listen :9338`. Wallet UI is then at `http://127.0.0.1:9338/wallet/`; the node's own explorer/wallet is at `http://127.0.0.1:8547/explorer/`.

### Funding a local account (faucet caveat)
- The testnet faucet (`POST /api/v1/faucet`) sends from the account derived from `ONEX_FAUCET_PRIVATE_KEY`. Without that env var set the endpoint returns `faucet disabled`, even with the `-faucet` flag.
- That faucet account must itself be funded. The simplest local setup: create a wallet (`onex wallet-create`), run the node with `-miner <that wallet's address>` so it earns block rewards, and set `ONEX_FAUCET_PRIVATE_KEY` to that wallet's private key. Then mining funds the miner/faucet account, and the faucet can drip to any new address.
- Block reward on testnet is 10 ONEX/block (see `configs/genesis-testnet.json`); a few seconds of mining is enough to fund sends.

### Wallet/addresses
- OneX uses Ed25519, 64-hex-char addresses (NOT Ethereum/MetaMask-compatible for signing). Wallet JSON (`onex wallet-create -out`) holds `address` + `privateKey` (private key = 64-byte hex = seed+pubkey).
