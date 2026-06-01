# Publish OneX Wallet (Android & iOS)

## Prerequisites

- [Expo account](https://expo.dev) and EAS CLI: `npm i -g eas-cli`
- **Apple Developer** + **Google Play Console** (you indicated both are ready)
- Production wallet URL: `https://YOUR_DOMAIN/wallet/` (see [DEPLOY.md](../DEPLOY.md))
- **GitHub Pages UI**: `https://zaragoza444.github.io/onex-blockchain/wallet/` + bridge URL in [docs/wallet/config.js](../docs/wallet/config.js) (see [docs/HOSTING.md](../docs/HOSTING.md))

## 1. Configure

```bash
cd mobile
cp .env.example .env
# Set EXPO_PUBLIC_WALLET_URL=https://YOUR_DOMAIN/wallet/
```

Link EAS project (first time):

```bash
eas login
eas init
# Update app.json extra.eas.projectId with the real ID from eas init
```

Edit `eas.json` submit section with your Apple ID and App Store Connect app ID.

## 2. Build store binaries

```bash
cd mobile
npm install
eas build --platform android --profile production
eas build --platform ios --profile production
```

Preview APK (internal testing):

```bash
eas build --platform android --profile preview
```

## 3. Google Play

1. Play Console → Create app → **OneX Wallet**
2. Package name: `com.onex.wallet` (must match `app.json`)
3. Upload **AAB** from EAS build
4. Complete: privacy policy URL, Data safety (network used), content rating
5. Internal testing track → promote to production

```bash
eas submit --platform android --profile production
```

## 4. Apple App Store

1. App Store Connect → New app → bundle ID `com.onex.wallet`
2. Upload **IPA** from EAS (or `eas submit --platform ios`)
3. App Privacy: WebView loads your wallet URL; declare network use
4. Export compliance: standard encryption exemption if applicable
5. Submit for review

```bash
eas submit --platform ios --profile production
```

## 5. Deep links

- `onexwallet://swap` → Swap tab
- `onexwallet://ai` → AI tab
- `onexwallet://earn` → Earn tab

## 6. Backend checklist

- TLS on `YOUR_DOMAIN`
- `ONEX_CORS_ORIGINS=https://YOUR_DOMAIN`
- `ONEX_API_KEY` set for write endpoints
- `docker compose -f docker-compose.prod.yml --profile proxy up -d`

## Local dev

| Platform | Wallet URL in Settings |
|----------|------------------------|
| iOS Simulator | `http://127.0.0.1:9338/wallet/` |
| Android emulator | `http://10.0.2.2:9338/wallet/` |
| Physical device (LAN) | `http://YOUR_PC_IP:9338/wallet/` |

Run bridge: `run-onex-wallet.bat` from repo root.
