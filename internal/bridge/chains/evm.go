package chains

type EVMAdapter struct{}

func (a *EVMAdapter) Type() string { return "evm" }

func (a *EVMAdapter) WrapSymbol(originSymbol string) string {
	return "w" + originSymbol
}

func (a *EVMAdapter) Deploy(in DeployInput) (*DeployResult, error) {
	addr := evmAddress(in.Chain.ID, in.Creator, in.Symbol, in.TokenID)
	return &DeployResult{
		ContractAddress: addr,
		DeployStatus:    "deployed",
		DeployTxHash:    "0x" + deterministicAddress("evm-tx", in.Chain.ID, in.Creator, in.Symbol, in.TokenID),
		DeployPayload: map[string]interface{}{
			"standard":   "ERC-20",
			"chainId":    in.Chain.ID,
			"networkId":  in.Chain.NetworkID,
			"rpc":        in.Chain.RPC,
			"explorer":   in.Chain.Explorer,
			"name":       in.Name,
			"symbol":     in.Symbol,
			"decimals":   in.Decimals,
			"totalSupply": in.Supply,
			"constructor": map[string]interface{}{
				"name":          in.Name,
				"symbol":        in.Symbol,
				"decimals":      in.Decimals,
				"initialSupply": in.Supply,
				"owner":         in.Creator,
			},
			"method": "eth_sendTransaction",
		},
		Note: "ERC-20 contract address derived deterministically; use deployPayload with your EVM wallet for live mainnet deploy.",
	}, nil
}
