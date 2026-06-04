package chains

import "fmt"

// For returns the chain adapter for the given chain type.
func For(chainType string) (Adapter, error) {
	switch chainType {
	case "onex":
		return &OneXAdapter{}, nil
	case "evm":
		return &EVMAdapter{}, nil
	case "solana":
		return &SolanaAdapter{}, nil
	case "btc", "tron":
		return &GenericAdapter{chainType: chainType}, nil
	default:
		return nil, fmt.Errorf("unsupported chain type: %s", chainType)
	}
}
