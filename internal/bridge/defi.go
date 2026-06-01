package bridge

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
	"github.com/onex-blockchain/onex/internal/rpc"
	"github.com/onex-blockchain/onex/internal/types"
)

func (b *Bridge) projectRoot() string {
	if b.cfg.ProjectRoot != "" {
		return b.cfg.ProjectRoot
	}
	for _, p := range []string{".", ".."} {
		if _, err := os.Stat(filepath.Join(p, "configs", "chains.json")); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return legacy.HomeDir()
}

func (b *Bridge) registry() *Registry {
	if b.reg == nil {
		b.reg = NewRegistry(b.projectRoot())
	}
	return b.reg
}

func (b *Bridge) portfolio() *PortfolioStore {
	if b.store == nil {
		b.store = NewPortfolioStore(filepath.Join(legacy.HomeDir(), "portfolios"))
	}
	return b.store
}

func (b *Bridge) syncOneXBalance(p *Portfolio) error {
	if err := b.EnsureWallet(); err != nil {
		return err
	}
	b.mu.RLock()
	addr := b.wallet.Address
	b.mu.RUnlock()
	bal, _, err := b.node.Balance(addr)
	if err != nil {
		return err
	}
	p.SetBalance(b.registry().TokenKey("onex-mainnet-1", "ONEX"), bal)
	return nil
}

func (b *Bridge) AllTokens(chainID string) []TokenInfo {
	b.mergeCustomTokensIntoRegistry()
	return b.registry().GetTokens(chainID)
}

func (b *Bridge) GetPortfolio() (*Portfolio, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	addr := b.WalletAddress()
	p, err := b.portfolio().Load(addr)
	if err != nil {
		return nil, err
	}
	_ = b.syncOneXBalance(p)
	_ = b.portfolio().Save(p)
	return p, nil
}

func (b *Bridge) DepositInfo(chainID string) (map[string]interface{}, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	addr := b.WalletAddress()
	reg := b.registry()
	var chain *ChainInfo
	for _, c := range reg.GetChains() {
		if c.ID == chainID {
			ch := c
			chain = &ch
			break
		}
	}
	if chain == nil {
		return nil, fmt.Errorf("unknown chain")
	}
	out := map[string]interface{}{
		"chain":           chain,
		"depositAddress":  DepositAddress(addr, chainID),
		"onexAddress":    addr,
		"note":            "Cross-chain deposits are credited to your portfolio after bridge confirmation.",
	}
	if chain.Type == "onex" {
		out["depositAddress"] = addr
		out["note"] = "Send ONEX directly to your OneX address on-chain."
	}
	return out, nil
}

func (b *Bridge) RecordDeposit(chainID, tokenID, amountStr, txHash string) (*DepositRecord, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	amt, err := rpc.ParseAmount(amountStr)
	if err != nil {
		return nil, err
	}
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	key := b.registry().TokenKey(chainID, tokenID)
	rec := DepositRecord{
		ID: newID(), ChainID: chainID, TokenID: tokenID,
		Amount: fmt.Sprintf("%d", amt), TxHash: txHash,
		Status: "pending", CreatedAt: nowUnix(),
	}
	for _, c := range b.registry().GetChains() {
		if c.ID == chainID && c.Type == "onex" {
			rec.Status = "confirmed"
			p.AddBalance(key, amt)
			break
		}
	}
	p.Deposits = append([]DepositRecord{rec}, p.Deposits...)
	if err := b.portfolio().Save(p); err != nil {
		return nil, err
	}
	b.completeTask(p, "deposit")
	return &rec, nil
}

func (b *Bridge) SwapQuote(fromChain, fromToken, toChain, toToken, amountStr string) (map[string]interface{}, error) {
	fromKey := b.registry().TokenKey(fromChain, fromToken)
	toKey := b.registry().TokenKey(toChain, toToken)
	if _, ok := b.ammStore().FindPool(fromKey, toKey); ok {
		return b.OneXSwapQuote(fromKey, toKey, amountStr)
	}
	amt, err := rpc.ParseAmount(amountStr)
	if err != nil {
		return nil, err
	}
	rate, ok := b.registry().SwapRate(fromKey, toKey)
	if !ok {
		return nil, fmt.Errorf("no swap pair for %s -> %s", fromKey, toKey)
	}
	fromHuman := float64(amt) / 1e8
	toHuman := fromHuman * rate
	toAtomic := uint64(math.Floor(toHuman * 1e8))
	return map[string]interface{}{
		"fromKey": fromKey, "toKey": toKey,
		"fromAmount": fmt.Sprintf("%d", amt),
		"toAmount":   fmt.Sprintf("%d", toAtomic),
		"rate":       rate,
	}, nil
}

func (b *Bridge) SwapExecute(fromChain, fromToken, toChain, toToken, amountStr string) (*SwapRecord, error) {
	fromKey := b.registry().TokenKey(fromChain, fromToken)
	toKey := b.registry().TokenKey(toChain, toToken)
	if _, ok := b.ammStore().FindPool(fromKey, toKey); ok {
		res, err := b.OneXSwapExecute(fromKey, toKey, amountStr, 50)
		if err != nil {
			return nil, err
		}
		return &SwapRecord{
			ID: newID(), FromKey: fromKey, ToKey: toKey,
			FromAmt: fmt.Sprintf("%v", res["amountIn"]), ToAmt: fmt.Sprintf("%v", res["amountOut"]),
			Status: "amm", CreatedAt: nowUnix(),
		}, nil
	}
	q, err := b.SwapQuote(fromChain, fromToken, toChain, toToken, amountStr)
	if err != nil {
		return nil, err
	}
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	fromKey = q["fromKey"].(string)
	toKey = q["toKey"].(string)
	var fromAmt uint64
	fmt.Sscanf(q["fromAmount"].(string), "%d", &fromAmt)
	var toAmt uint64
	fmt.Sscanf(q["toAmount"].(string), "%d", &toAmt)

	if err := p.SubBalance(fromKey, fromAmt); err != nil {
		return nil, err
	}
	p.AddBalance(toKey, toAmt)

	rec := SwapRecord{
		ID: newID(), FromKey: fromKey, ToKey: toKey,
		FromAmt: q["fromAmount"].(string), ToAmt: q["toAmount"].(string),
		Status: "completed", CreatedAt: nowUnix(),
	}
	p.Swaps = append([]SwapRecord{rec}, p.Swaps...)
	_ = b.portfolio().Save(p)
	b.completeTask(p, "first-swap")
	return &rec, nil
}

func (b *Bridge) MintNFT(name, description, imageURL, chainID string) (*NFTAsset, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	nft := NFTAsset{
		ID: newID(), ChainID: chainID, Name: name,
		Description: description, ImageURL: imageURL,
		Owner: b.WalletAddress(), CreatedAt: nowUnix(),
	}
	p.NFTs = append([]NFTAsset{nft}, p.NFTs...)
	_ = b.portfolio().Save(p)
	b.completeTask(p, "mint-nft")
	return &nft, nil
}

func (b *Bridge) TransferNFT(id, to string) error {
	p, err := b.GetPortfolio()
	if err != nil {
		return err
	}
	for i := range p.NFTs {
		if p.NFTs[i].ID == id {
			p.NFTs[i].Owner = to
			return b.portfolio().Save(p)
		}
	}
	return fmt.Errorf("nft not found")
}

func (b *Bridge) CompleteTask(taskID string) error {
	p, err := b.GetPortfolio()
	if err != nil {
		return err
	}
	for i := range p.Tasks {
		if p.Tasks[i].ID == taskID && p.Tasks[i].Status == "open" {
			p.Tasks[i].Status = "done"
			var reward uint64
			fmt.Sscanf(p.Tasks[i].RewardAmt, "%d", &reward)
			p.AddBalance(p.Tasks[i].RewardKey, reward)
			_ = b.portfolio().Save(p)
			return nil
		}
	}
	return fmt.Errorf("task not found or already done")
}

func (b *Bridge) completeTask(p *Portfolio, id string) {
	for i := range p.Tasks {
		if p.Tasks[i].ID == id && p.Tasks[i].Status == "open" {
			p.Tasks[i].Status = "done"
			var reward uint64
			fmt.Sscanf(p.Tasks[i].RewardAmt, "%d", &reward)
			p.AddBalance(p.Tasks[i].RewardKey, reward)
			_ = b.portfolio().Save(p)
			return
		}
	}
}

func parseAtomicAmount(s string) (uint64, error) {
	if strings.Contains(s, ".") {
		return rpc.ParseAmount(s)
	}
	var n uint64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}

func (b *Bridge) CreateLoan(loanType, collateralKey, collateralAmt, debtKey, debtAmt string, apy float64) (*LoanRecord, error) {
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	col, err := parseAtomicAmount(collateralAmt)
	if err != nil {
		return nil, err
	}
	if err := p.SubBalance(collateralKey, col); err != nil {
		return nil, err
	}
	debt, err := parseAtomicAmount(debtAmt)
	if err != nil {
		return nil, err
	}
	if loanType == "borrow" {
		p.AddBalance(debtKey, debt)
	}
	loan := LoanRecord{
		ID: newID(), Type: loanType,
		CollateralKey: collateralKey, CollateralAmt: fmt.Sprintf("%d", col),
		DebtKey: debtKey, DebtAmt: fmt.Sprintf("%d", debt),
		APY: apy, Status: "active", CreatedAt: nowUnix(),
	}
	p.Loans = append([]LoanRecord{loan}, p.Loans...)
	_ = b.portfolio().Save(p)
	return &loan, nil
}

func (b *Bridge) RepayLoan(id string) error {
	p, err := b.GetPortfolio()
	if err != nil {
		return err
	}
	for i := range p.Loans {
		if p.Loans[i].ID == id && p.Loans[i].Status == "active" {
			var debt uint64
			fmt.Sscanf(p.Loans[i].DebtAmt, "%d", &debt)
			if err := p.SubBalance(p.Loans[i].DebtKey, debt); err != nil {
				return err
			}
			var col uint64
			fmt.Sscanf(p.Loans[i].CollateralAmt, "%d", &col)
			p.AddBalance(p.Loans[i].CollateralKey, col)
			p.Loans[i].Status = "repaid"
			return b.portfolio().Save(p)
		}
	}
	return fmt.Errorf("loan not found")
}

func (b *Bridge) SendToken(chainID, tokenID, toAddr, amountStr, feeStr string) (map[string]string, error) {
	if chainID == "onex-mainnet-1" && tokenID == "ONEX" {
		amount, err := rpc.ParseAmount(amountStr)
		if err != nil {
			return nil, err
		}
		fee, _ := rpc.ParseAmount(feeStr)
		if feeStr == "" {
			fee, _ = rpc.ParseAmount("0.001")
		}
		return b.Send(types.Address(toAddr), amount, fee)
	}
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	key := b.registry().TokenKey(chainID, tokenID)
	amt, err := rpc.ParseAmount(amountStr)
	if err != nil {
		return nil, err
	}
	if err := p.SubBalance(key, amt); err != nil {
		return nil, err
	}
	_ = b.portfolio().Save(p)
	return map[string]string{"status": "sent-offchain", "token": key}, nil
}

func nowUnix() int64 {
	return time.Now().Unix()
}
