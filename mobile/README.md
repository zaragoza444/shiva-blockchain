# OneX Wallet (mobile)

Expo React Native app that loads the OKX-style OneX Wallet UI in a WebView.

## Quick start

```bash
# Terminal 1 — node + bridge
cd ..
run-onex-wallet.bat

# Terminal 2 — mobile
cd mobile
cp .env.example .env
npm install
npm start
```

Scan the QR code with Expo Go, or press `a` / `i` for emulator.

## Configuration

| Variable | Description |
|----------|-------------|
| `EXPO_PUBLIC_WALLET_URL` | Wallet URL (default `http://127.0.0.1:9338/wallet/`) |

Override in-app via **Settings** (gear icon).

## Deep links

- `onexwallet://swap`
- `onexwallet://ai`
- `onexwallet://earn`
- `onexwallet://discover`
- `onexwallet://web3`

## Store builds

See [PUBLISH.md](PUBLISH.md).
