package chains

// DeployChain is chain metadata passed into adapters (avoids importing bridge).
type DeployChain struct {
	ID        string
	Name      string
	NetworkID uint64
	RPC       string
	Explorer  string
	Type      string
}

// DeployInput describes a token launch on a specific chain.
type DeployInput struct {
	Chain    DeployChain
	Name     string
	Symbol   string
	Decimals int
	Supply   uint64
	Creator  string
	TokenID  string
}

// DeployResult holds on-chain deployment metadata returned by a chain adapter.
type DeployResult struct {
	ContractAddress string                 `json:"contractAddress"`
	DeployStatus    string                 `json:"deployStatus"`
	DeployTxHash    string                 `json:"deployTxHash,omitempty"`
	DeployPayload   map[string]interface{} `json:"deployPayload,omitempty"`
	Note            string                 `json:"note,omitempty"`
}

// Adapter deploys tokens on a specific chain family.
type Adapter interface {
	Type() string
	Deploy(in DeployInput) (*DeployResult, error)
	WrapSymbol(originSymbol string) string
}
