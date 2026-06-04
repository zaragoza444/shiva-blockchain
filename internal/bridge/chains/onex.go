package chains

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type OneXAdapter struct{}

func (a *OneXAdapter) Type() string { return "onex" }

func (a *OneXAdapter) WrapSymbol(originSymbol string) string {
	return "w" + originSymbol
}

func (a *OneXAdapter) Deploy(in DeployInput) (*DeployResult, error) {
	regHash := sha256.Sum256([]byte(fmt.Sprintf("onex-token:%s:%s:%s", in.Chain.ID, in.TokenID, in.Creator)))
	return &DeployResult{
		ContractAddress: hex.EncodeToString(regHash[:]),
		DeployStatus:    "registered",
		DeployPayload: map[string]interface{}{
			"standard":  "onex-native-token-v1",
			"chainId":   in.Chain.ID,
			"tokenId":   in.TokenID,
			"name":      in.Name,
			"symbol":    in.Symbol,
			"decimals":  in.Decimals,
			"supply":    in.Supply,
			"creator":   in.Creator,
			"networkId": in.Chain.NetworkID,
		},
		Note: "Token registered on OneX ledger; balances tracked in portfolio and bridge registry.",
	}, nil
}
