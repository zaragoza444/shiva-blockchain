package chains

type SolanaAdapter struct{}

func (a *SolanaAdapter) Type() string { return "solana" }

func (a *SolanaAdapter) WrapSymbol(originSymbol string) string {
	return "w" + originSymbol
}

func (a *SolanaAdapter) Deploy(in DeployInput) (*DeployResult, error) {
	mint := solanaMint(in.Chain.ID, in.Creator, in.Symbol, in.TokenID)
	return &DeployResult{
		ContractAddress: mint,
		DeployStatus:    "deployed",
		DeployTxHash:    deterministicAddress("sol-tx", in.Chain.ID, in.Creator, in.Symbol, in.TokenID),
		DeployPayload: map[string]interface{}{
			"standard":  "SPL",
			"chainId":   in.Chain.ID,
			"rpc":       in.Chain.RPC,
			"explorer":  in.Chain.Explorer,
			"mint":      mint,
			"name":      in.Name,
			"symbol":    in.Symbol,
			"decimals":  in.Decimals,
			"supply":    in.Supply,
			"authority": in.Creator,
		},
		Note: "SPL mint address generated; sign with Phantom/Solflare using deployPayload for live mint.",
	}, nil
}
