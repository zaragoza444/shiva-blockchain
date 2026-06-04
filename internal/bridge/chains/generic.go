package chains

type GenericAdapter struct {
	chainType string
}

func (a *GenericAdapter) Type() string { return a.chainType }

func (a *GenericAdapter) WrapSymbol(originSymbol string) string {
	return "w" + originSymbol
}

func (a *GenericAdapter) Deploy(in DeployInput) (*DeployResult, error) {
	var addr string
	switch a.chainType {
	case "btc":
		addr = btcTokenRef(in.Chain.ID, in.Creator, in.Symbol, in.TokenID)
	case "tron":
		addr = tronAddress(in.Chain.ID, in.Creator, in.Symbol, in.TokenID)
	default:
		addr = deterministicAddress(a.chainType, in.Chain.ID, in.Creator, in.Symbol, in.TokenID)
	}
	return &DeployResult{
		ContractAddress: addr,
		DeployStatus:    "registered",
		DeployPayload: map[string]interface{}{
			"standard":  a.chainType + "-asset-v1",
			"chainId":   in.Chain.ID,
			"symbol":    in.Symbol,
			"decimals":  in.Decimals,
			"supply":    in.Supply,
			"reference": addr,
		},
		Note: "Asset registered in OneX Token Platform catalog for " + in.Chain.Name + ".",
	}, nil
}
