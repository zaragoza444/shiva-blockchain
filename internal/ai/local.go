package ai

import (
	"strings"
)

const walletSystemHint = `You are Shiva AI — assistant for Shiva Blockchain (Ed25519 PoW chain) and Shiva Wallet (OKX-style DeFi UI).
Features: multi-chain portfolio, send/receive, deposit, Shiva Swap AMM (x·y=k), liquidity pools, cross-chain bridge, stake, loans, NFTs, tasks, create token.
Native coin: SHIVA (8 decimals). Addresses are 64-char hex. MetaMask cannot sign; use Shiva Wallet or the Chrome extension.
Wallet UI tabs: Wallet (home), Trade (swap/pool/bridge), Earn (stake/loans), Discover (NFT/tasks/token/networks), Web3 (dApps), AI (this chat).
Node API: :8545 explorer, /rpc JSON-RPC, /health. Bridge: :9338/wallet/.`

func localReply(user string, ctx string) ChatResponse {
	q := strings.ToLower(strings.TrimSpace(user))
	reply := ""
	var act *Action
	suggestions := []string{"Show my balance", "How do I swap?", "Explain Shiva Swap", "Stake SHIVA"}

	switch {
	case containsAny(q, "hello", "hi", "hey"):
		reply = "Hello! I'm Shiva AI. I can help with your wallet, swaps, staking, bridge, and the Shiva blockchain. What would you like to do?"
	case containsAny(q, "balance", "portfolio", "assets", "how much"):
		reply = "Open the Wallet tab to see total assets and token rows. I use your live portfolio context when the bridge is connected."
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		}
		act = &Action{Type: "navigate", Tab: "wallet"}
	case containsAny(q, "send", "transfer"):
		reply = "Tap Send on the home screen (or the send sheet). Pick chain + token, enter a 64-char recipient address and amount. On-chain SHIVA sends need a small fee (default 0.001 SHIVA)."
		act = &Action{Type: "sheet", Sheet: "send"}
	case containsAny(q, "receive", "address"):
		reply = "Tap Receive to copy your Shiva address. Share it to receive SHIVA on the Shiva chain."
		act = &Action{Type: "sheet", Sheet: "receive"}
	case containsAny(q, "deposit"):
		reply = "Deposit credits portfolio tokens from other chains. Open Deposit, choose chain, copy the deposit address, then record the amount (and tx hash if you have one)."
		act = &Action{Type: "sheet", Sheet: "deposit"}
	case containsAny(q, "swap", "trade", "exchange"):
		reply = "Shiva Swap is a Uniswap-style AMM (constant product x·y=k, ~0.3% fee). Go to Trade → Swap, pick tokens and amount, review price impact, then confirm."
		act = &Action{Type: "navigate", Tab: "trade"}
	case containsAny(q, "pool", "liquidity", "lp"):
		reply = "Trade → Pool: add liquidity to AMM pairs and earn LP shares. Remove liquidity by burning shares."
		act = &Action{Type: "navigate", Tab: "trade"}
	case containsAny(q, "bridge", "cross-chain", "cross chain"):
		reply = "Trade → Bridge routes swaps across chains via SHIVA hub pools. Select from/to chain and token, quote, then bridge."
		act = &Action{Type: "navigate", Tab: "trade"}
	case containsAny(q, "stake", "staking", "apy", "earn"):
		reply = "Earn tab → Stake: lock tokens for APY and receipt tokens (e.g. sSHIVA). Check lock period before unstaking."
		act = &Action{Type: "navigate", Tab: "earn"}
	case containsAny(q, "loan", "borrow", "lend"):
		reply = "Earn tab → Loans: post collateral and borrow or lend against configured token pairs. Repay to close active loans."
		act = &Action{Type: "navigate", Tab: "earn"}
	case containsAny(q, "nft", "mint"):
		reply = "Discover → NFT: view your collection or mint with name, description, and image URL."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "task", "reward", "claim"):
		reply = "Discover → Rewards: complete open tasks to claim SHIVA or wSHIVA bonuses."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "create token", "mint token", "launch"):
		reply = "Discover → Create token: set name, symbol, decimals, and supply on a chosen chain. Tokens appear in your portfolio."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "network", "chain", "chains"):
		reply = "Discover → Networks lists 13+ supported chains (Shiva, Ethereum, BSC, Polygon, and more)."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "web3", "dapp", "explorer"):
		reply = "Web3 tab opens dApps and the block explorer. Shiva provider uses Ed25519; install the Chrome extension for dApp signing."
		act = &Action{Type: "navigate", Tab: "web3"}
	case containsAny(q, "metamask", "ethereum", "evm"):
		reply = "Shiva uses Ed25519, not Ethereum keys. MetaMask can display network info but cannot sign Shiva transactions. Use Shiva Wallet or the extension."
	case containsAny(q, "node", "blockchain", "pow", "mining", "block"):
		reply = "Shiva is a proof-of-work node (shivad) with REST + JSON-RPC. Explorer at /explorer/, health at /health. JSON-RPC includes shiva_* and eth_* compat methods."
	case containsAny(q, "rpc", "api", "json"):
		reply = "Node JSON-RPC: POST /rpc (e.g. shiva_getBalance, shiva_sendTransaction, eth_chainId). Bridge RPC: :9338/rpc for wallet methods."
	case containsAny(q, "wallet", "create", "import"):
		reply = "Create a wallet via Settings (⚙) or the + button. Wallet file: ~/.shiva/wallets/default.json. Ed25519 keys — keep backups offline."
		act = &Action{Type: "sheet", Sheet: "settings"}
	case containsAny(q, "fee", "gas"):
		reply = "Shiva uses explicit min tx fees (not EVM gas). Default send fee is 0.001 SHIVA. AMM swaps charge pool fee (~0.3%)."
	case containsAny(q, "cloud", "api key", "openai", "model"):
		reply = "Set SHIVA_AI_API_KEY (and optional SHIVA_AI_BASE_URL, SHIVA_AI_MODEL) on the bridge/node to enable cloud AI. Without a key, I run in local assistant mode."
	default:
		reply = walletSystemHint + "\n\nAsk about: balance, send, swap, stake, bridge, NFTs, loans, or how to run the node. "
		if ctx != "" {
			reply += "Here's your current context:\n" + summarizeContext(ctx)
		} else {
			reply += "Connect a wallet and refresh for personalized answers."
		}
	}

	return ChatResponse{
		Reply:       reply,
		Mode:        "local",
		Action:      act,
		Suggestions: suggestions,
	}
}

func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func summarizeContext(ctx string) string {
	if len(ctx) > 1200 {
		return ctx[:1200] + "…"
	}
	return ctx
}
