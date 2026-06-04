package chains

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func deterministicAddress(prefix, chainID, creator, symbol, tokenID string) string {
	h := sha256.Sum256([]byte(prefix + ":" + chainID + ":" + creator + ":" + symbol + ":" + tokenID))
	return hex.EncodeToString(h[:20])
}

func evmAddress(chainID, creator, symbol, tokenID string) string {
	return "0x" + deterministicAddress("evm", chainID, creator, symbol, tokenID)
}

func solanaMint(chainID, creator, symbol, tokenID string) string {
	raw := deterministicAddress("sol", chainID, creator, symbol, tokenID)
	return fmt.Sprintf("So1%s", raw[:40])
}

func tronAddress(chainID, creator, symbol, tokenID string) string {
	return "T" + deterministicAddress("trx", chainID, creator, symbol, tokenID)[:33]
}

func btcTokenRef(chainID, creator, symbol, tokenID string) string {
	return "btc1" + deterministicAddress("btc", chainID, creator, symbol, tokenID)[:32]
}
