package chains_test

import (
	"strings"
	"testing"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
)

func TestEVMDeploy(t *testing.T) {
	a := &chains.EVMAdapter{}
	res, err := a.Deploy(chains.DeployInput{
		Chain: chains.DeployChain{ID: "ethereum", Name: "Ethereum", NetworkID: 1, Type: "evm"},
		Name: "Test", Symbol: "TST", Decimals: 8, Supply: 1_000_000_000,
		Creator: "abc123", TokenID: "TST",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.ContractAddress, "0x") {
		t.Fatalf("expected 0x address, got %s", res.ContractAddress)
	}
	if res.DeployStatus != "deployed" {
		t.Fatalf("status %s", res.DeployStatus)
	}
}

func TestWrapSymbol(t *testing.T) {
	a := &chains.EVMAdapter{}
	if got := a.WrapSymbol("MYC"); got != "wMYC" {
		t.Fatalf("wrap symbol %s", got)
	}
}

func TestFactory(t *testing.T) {
	for _, typ := range []string{"onex", "evm", "solana", "btc", "tron"} {
		if _, err := chains.For(typ); err != nil {
			t.Fatalf("type %s: %v", typ, err)
		}
	}
}
