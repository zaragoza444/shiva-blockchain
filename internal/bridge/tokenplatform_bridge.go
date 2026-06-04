package bridge

import (
	"fmt"
	"strings"

	"github.com/onex-blockchain/onex/internal/bridge/chains"
	"github.com/onex-blockchain/onex/internal/bridge/tokenplatform"
	"github.com/onex-blockchain/onex/internal/rpc"
)

func (b *Bridge) platformStore() *tokenplatform.Store {
	if b.platform == nil {
		b.platform = tokenplatform.NewStore()
	}
	return b.platform
}

func (b *Bridge) platformConfig() tokenplatform.PlatformConfig {
	return tokenplatform.LoadConfig(b.projectRoot())
}

func (b *Bridge) findChain(chainID string) (*ChainInfo, error) {
	reg := b.registry()
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	for i := range reg.Chains {
		if reg.Chains[i].ID == chainID {
			ch := reg.Chains[i]
			return &ch, nil
		}
	}
	return nil, fmt.Errorf("unknown chain: %s", chainID)
}

func chainToDeployChain(c ChainInfo) chains.DeployChain {
	return chains.DeployChain{
		ID: c.ID, Name: c.Name, NetworkID: c.NetworkID,
		RPC: c.RPC, Explorer: c.Explorer, Type: c.Type,
	}
}

// PlatformStatus returns token platform overview.
func (b *Bridge) PlatformStatus() (tokenplatform.StatusSummary, error) {
	tokens, _ := b.ListPlatformTokens()
	wraps, _ := b.platformStore().LoadWraps()
	cfg := b.platformConfig()
	return tokenplatform.BuildStatus(cfg, tokens, wraps, len(b.registry().GetChains())), nil
}

// ListPlatformTokens returns all platform-deployed tokens (includes legacy custom tokens).
func (b *Bridge) ListPlatformTokens() ([]tokenplatform.PlatformToken, error) {
	b.mergeCustomTokensIntoRegistry()
	platform, _ := b.platformStore().LoadTokens()
	custom, _ := b.customTokens().load()
	types := b.chainTypeMap()
	lites := make([]tokenplatform.CustomTokenLite, 0, len(custom))
	for _, ct := range custom {
		lites = append(lites, tokenplatform.CustomTokenLite{
			ID: ct.ID, ChainID: ct.ChainID, Name: ct.Name, Symbol: ct.Symbol,
			Decimals: ct.Decimals, Supply: ct.Supply, Creator: ct.Creator, CreatedAt: ct.CreatedAt,
		})
	}
	return tokenplatform.MergeWithCustom(platform, lites, types), nil
}

func (b *Bridge) chainTypeMap() map[string]string {
	out := make(map[string]string)
	for _, c := range b.registry().GetChains() {
		out[c.ID] = c.Type
	}
	return out
}

// GetPlatformToken returns one token by chain and id.
func (b *Bridge) GetPlatformToken(chainID, tokenID string) (*tokenplatform.PlatformToken, error) {
	list, err := b.ListPlatformTokens()
	if err != nil {
		return nil, err
	}
	t, _ := tokenplatform.FindToken(list, chainID, tokenID)
	if t == nil {
		return nil, fmt.Errorf("token not found")
	}
	return t, nil
}

// DeployPlatformToken launches a token on any supported chain with on-chain metadata.
func (b *Bridge) DeployPlatformToken(chainID, name, symbol string, decimals int, supplyStr string) (*tokenplatform.PlatformToken, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	if name == "" || symbol == "" {
		return nil, fmt.Errorf("name and symbol required")
	}
	if decimals < 0 || decimals > 18 {
		return nil, fmt.Errorf("decimals must be 0-18")
	}
	supply, err := rpc.ParseAmount(supplyStr)
	if err != nil {
		return nil, err
	}
	if supply == 0 {
		return nil, fmt.Errorf("supply must be > 0")
	}

	chain, err := b.findChain(chainID)
	if err != nil {
		return nil, err
	}
	adapter, err := chains.For(chain.Type)
	if err != nil {
		return nil, err
	}

	store := b.platformStore()
	tokens, err := store.LoadTokens()
	if err != nil {
		return nil, err
	}
	tokenID := tokenplatform.UniqueTokenID(tokens, chainID, tokenplatform.SanitizeTokenID(symbol))

	deploy, err := adapter.Deploy(chains.DeployInput{
		Chain: chainToDeployChain(*chain), Name: name, Symbol: symbol,
		Decimals: decimals, Supply: supply, Creator: b.WalletAddress(), TokenID: tokenID,
	})
	if err != nil {
		return nil, err
	}

	pt := tokenplatform.PlatformToken{
		ID: tokenID, ChainID: chainID, ChainType: chain.Type,
		Name: name, Symbol: strings.ToUpper(symbol), Decimals: decimals,
		Supply: fmt.Sprintf("%d", supply), Creator: b.WalletAddress(),
		CreatedAt: tokenplatform.NowUnix(),
		ContractAddress: deploy.ContractAddress, DeployStatus: deploy.DeployStatus,
		DeployTxHash: deploy.DeployTxHash, DeployPayload: deploy.DeployPayload,
	}
	tokens = append(tokens, pt)
	if err := store.SaveTokens(tokens); err != nil {
		return nil, err
	}

	if err := b.registerPlatformToken(pt); err != nil {
		return nil, err
	}
	b.completeTaskAfterDeploy()
	return &pt, nil
}

func (b *Bridge) registerPlatformToken(pt tokenplatform.PlatformToken) error {
	ct := CustomToken{
		ID: pt.ID, ChainID: pt.ChainID, Name: pt.Name, Symbol: pt.Symbol,
		Decimals: pt.Decimals, Supply: pt.Supply, Creator: pt.Creator, CreatedAt: pt.CreatedAt,
	}
	tokens, _ := b.customTokens().load()
	dup := false
	for _, t := range tokens {
		if t.ChainID == ct.ChainID && t.ID == ct.ID {
			dup = true
			break
		}
	}
	if !dup {
		tokens = append(tokens, ct)
		if err := b.customTokens().save(tokens); err != nil {
			return err
		}
	}
	b.mergeCustomTokensIntoRegistry()

	p, err := b.GetPortfolio()
	if err != nil {
		return err
	}
	key := b.registry().TokenKey(pt.ChainID, pt.ID)
	p.AddBalance(key, mustParseUint(pt.Supply))
	if p.CreatedTokens == nil {
		p.CreatedTokens = []string{}
	}
	p.CreatedTokens = append(p.CreatedTokens, key)
	return b.portfolio().Save(p)
}

func mustParseUint(s string) uint64 {
	var n uint64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func (b *Bridge) completeTaskAfterDeploy() {
	p, err := b.GetPortfolio()
	if err != nil {
		return
	}
	b.completeTask(p, "create-token")
}

// WrapPlatformToken locks origin supply and mints wrapped token on target chain.
func (b *Bridge) WrapPlatformToken(originChainID, originTokenID, targetChainID, amountStr string) (*tokenplatform.WrapRecord, *tokenplatform.PlatformToken, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, nil, err
	}
	amount, err := rpc.ParseAmount(amountStr)
	if err != nil {
		return nil, nil, err
	}
	if err := tokenplatform.ValidateWrapAmount(amount); err != nil {
		return nil, nil, err
	}

	originChain, err := b.findChain(originChainID)
	if err != nil {
		return nil, nil, err
	}
	targetChain, err := b.findChain(targetChainID)
	if err != nil {
		return nil, nil, err
	}
	if originChainID == targetChainID {
		return nil, nil, fmt.Errorf("origin and target chain must differ")
	}

	store := b.platformStore()
	tokens, err := store.LoadTokens()
	if err != nil {
		return nil, nil, err
	}
	origin, originIdx := tokenplatform.FindToken(tokens, originChainID, originTokenID)
	if origin == nil {
		// allow wrapping registry tokens not created via platform
		regTok := b.registry().FindToken(originChainID, originTokenID)
		if regTok == nil {
			return nil, nil, fmt.Errorf("origin token not found")
		}
		origin = &tokenplatform.PlatformToken{
			ID: regTok.ID, ChainID: regTok.ChainID, Name: regTok.Name,
			Symbol: regTok.Symbol, Decimals: regTok.Decimals, ChainType: originChain.Type,
		}
		originIdx = -1
	}

	p, err := b.GetPortfolio()
	if err != nil {
		return nil, nil, err
	}
	originKey := b.registry().TokenKey(originChainID, originTokenID)
	if p.GetBalance(originKey) < amount {
		return nil, nil, fmt.Errorf("insufficient balance to wrap")
	}

	adapter, err := chains.For(targetChain.Type)
	if err != nil {
		return nil, nil, err
	}
	wrappedSymbol := adapter.WrapSymbol(origin.Symbol)
	wrappedID := tokenplatform.WrapTokenID(originTokenID, targetChainID)

	// reuse existing wrapped token on target chain if present
	var wrapped *tokenplatform.PlatformToken
	for i := range tokens {
		if tokens[i].ChainID == targetChainID && tokens[i].OriginKey == originKey {
			wrapped = &tokens[i]
			break
		}
	}

	if wrapped == nil {
		deploy, err := adapter.Deploy(chains.DeployInput{
			Chain: chainToDeployChain(*targetChain), Name: origin.Name + " (Wrapped)",
			Symbol: wrappedSymbol, Decimals: origin.Decimals, Supply: amount,
			Creator: b.WalletAddress(), TokenID: wrappedID,
		})
		if err != nil {
			return nil, nil, err
		}
		wt := tokenplatform.PlatformToken{
			ID: wrappedID, ChainID: targetChainID, ChainType: targetChain.Type,
			Name: origin.Name + " (Wrapped)", Symbol: wrappedSymbol, Decimals: origin.Decimals,
			Supply: fmt.Sprintf("%d", amount), Creator: b.WalletAddress(),
			CreatedAt: tokenplatform.NowUnix(), OriginKey: originKey, IsWrapped: true,
			ContractAddress: deploy.ContractAddress, DeployStatus: deploy.DeployStatus,
			DeployTxHash: deploy.DeployTxHash, DeployPayload: deploy.DeployPayload,
		}
		tokens = append(tokens, wt)
		wrapped = &wt

		ct := CustomToken{
			ID: wt.ID, ChainID: wt.ChainID, Name: wt.Name, Symbol: wt.Symbol,
			Decimals: wt.Decimals, Supply: "0", Creator: wt.Creator, CreatedAt: wt.CreatedAt,
		}
		custom, _ := b.customTokens().load()
		custom = append(custom, ct)
		_ = b.customTokens().save(custom)
	} else {
		supply := mustParseUint(wrapped.Supply) + amount
		for i := range tokens {
			if tokens[i].ChainID == targetChainID && tokens[i].ID == wrapped.ID {
				tokens[i].Supply = fmt.Sprintf("%d", supply)
				wrapped = &tokens[i]
				break
			}
		}
	}

	wrapID := tokenplatform.SanitizeTokenID("W" + newID())
	rec := tokenplatform.WrapRecord{
		ID: wrapID, OriginKey: originKey, TargetChainID: targetChainID,
		WrappedTokenID: wrapped.ID, WrappedSymbol: wrapped.Symbol,
		Amount: fmt.Sprintf("%d", amount), Status: "completed",
		CreatedAt: tokenplatform.NowUnix(),
		BridgeTxHash: "0x" + newID(),
	}

	wraps, _ := store.LoadWraps()
	wraps = append(wraps, rec)
	if originIdx >= 0 {
		tokens = tokenplatform.AppendWrapRecord(tokens, originIdx, rec)
	}
	if err := store.SaveTokens(tokens); err != nil {
		return nil, nil, err
	}
	if err := store.SaveWraps(wraps); err != nil {
		return nil, nil, err
	}

	b.mergeCustomTokensIntoRegistry()
	if err := p.SubBalance(originKey, amount); err != nil {
		return nil, nil, err
	}
	wrappedKey := b.registry().TokenKey(targetChainID, wrapped.ID)
	p.AddBalance(wrappedKey, amount)
	_ = b.portfolio().Save(p)

	return &rec, wrapped, nil
}

// ListWrapRecords returns cross-chain wrap history.
func (b *Bridge) ListWrapRecords() ([]tokenplatform.WrapRecord, error) {
	return b.platformStore().LoadWraps()
}

// CreateToken delegates to the platform deploy (backward compatible).
func (b *Bridge) CreateTokenViaPlatform(chainID, name, symbol string, decimals int, supplyStr string) (*CustomToken, error) {
	pt, err := b.DeployPlatformToken(chainID, name, symbol, decimals, supplyStr)
	if err != nil {
		return nil, err
	}
	return &CustomToken{
		ID: pt.ID, ChainID: pt.ChainID, Name: pt.Name, Symbol: pt.Symbol,
		Decimals: pt.Decimals, Supply: pt.Supply, Creator: pt.Creator, CreatedAt: pt.CreatedAt,
	}, nil
}
