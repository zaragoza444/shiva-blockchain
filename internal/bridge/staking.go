package bridge

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/onex-blockchain/onex/internal/rpc"
)

type StakePool struct {
	ID           string  `json:"id"`
	ChainID      string  `json:"chainId"`
	StakeToken   string  `json:"stakeToken"`
	ReceiptToken string  `json:"receiptToken"`
	APY          float64 `json:"apy"`
	LockDays     int     `json:"lockDays"`
	MinStake     string  `json:"minStake"`
}

type StakeRecord struct {
	ID           string  `json:"id"`
	PoolID       string  `json:"poolId"`
	StakeKey     string  `json:"stakeKey"`
	ReceiptKey   string  `json:"receiptKey"`
	Amount       string  `json:"amount"`
	ReceiptAmt   string  `json:"receiptAmount"`
	APY          float64 `json:"apy"`
	StakedAt     int64   `json:"stakedAt"`
	UnlockAt     int64   `json:"unlockAt"`
	Status       string  `json:"status"` // active, unstaked
}

func (b *Bridge) stakePools() []StakePool {
	pools := loadJSON[StakePool](filepath.Join(b.projectRoot(), "configs", "stake-pools.json"))
	// ensure receipt tokens exist in registry for display
	for _, p := range pools {
		b.ensureReceiptToken(p)
	}
	return pools
}

func (b *Bridge) ensureReceiptToken(p StakePool) {
	b.registry().appendToken(TokenInfo{
		ID: p.ReceiptToken, ChainID: p.ChainID,
		Name: "Staked " + p.StakeToken, Symbol: p.ReceiptToken, Decimals: 8,
	})
}

func (b *Bridge) findStakePool(id string) (*StakePool, error) {
	for _, p := range b.stakePools() {
		if p.ID == id {
			cp := p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("stake pool not found")
}

func (b *Bridge) ListStakes() ([]StakeRecord, error) {
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	return p.Stakes, nil
}

func (b *Bridge) Stake(poolID, amountStr string) (*StakeRecord, error) {
	pool, err := b.findStakePool(poolID)
	if err != nil {
		return nil, err
	}
	amt, err := rpc.ParseAmount(amountStr)
	if err != nil {
		return nil, err
	}
	minAmt, _ := rpc.ParseAmount(pool.MinStake)
	if amt < minAmt {
		return nil, fmt.Errorf("minimum stake is %s %s", pool.MinStake, pool.StakeToken)
	}
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	stakeKey := b.registry().TokenKey(pool.ChainID, pool.StakeToken)
	receiptKey := b.registry().TokenKey(pool.ChainID, pool.ReceiptToken)
	if err := p.SubBalance(stakeKey, amt); err != nil {
		return nil, err
	}
	// receipt 1:1 + small staking bonus (0.1% on stake)
	bonus := amt / 1000
	receiptTotal := amt + bonus
	p.AddBalance(receiptKey, receiptTotal)
	now := time.Now().Unix()
	unlock := now
	if pool.LockDays > 0 {
		unlock = now + int64(pool.LockDays)*86400
	}
	rec := StakeRecord{
		ID: newID(), PoolID: poolID, StakeKey: stakeKey, ReceiptKey: receiptKey,
		Amount: fmt.Sprintf("%d", amt), ReceiptAmt: fmt.Sprintf("%d", receiptTotal),
		APY: pool.APY, StakedAt: now, UnlockAt: unlock, Status: "active",
	}
	p.Stakes = append([]StakeRecord{rec}, p.Stakes...)
	_ = b.portfolio().Save(p)
	b.completeTask(p, "first-stake")
	return &rec, nil
}

func (b *Bridge) Unstake(stakeID string) (map[string]string, error) {
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	for i := range p.Stakes {
		s := &p.Stakes[i]
		if s.ID != stakeID || s.Status != "active" {
			continue
		}
		if now < s.UnlockAt {
			return nil, fmt.Errorf("locked until %s", time.Unix(s.UnlockAt, 0).Format(time.RFC822))
		}
		var receipt uint64
		fmt.Sscanf(s.ReceiptAmt, "%d", &receipt)
		var principal uint64
		fmt.Sscanf(s.Amount, "%d", &principal)
		if err := p.SubBalance(s.ReceiptKey, receipt); err != nil {
			return nil, err
		}
		// return principal + pro-rata rewards estimate
		days := float64(now-s.StakedAt) / 86400
		reward := uint64(float64(principal) * (s.APY / 100) * days / 365)
		p.AddBalance(s.StakeKey, principal+reward)
		s.Status = "unstaked"
		_ = b.portfolio().Save(p)
		return map[string]string{
			"status":   "unstaked",
			"returned": fmt.Sprintf("%d", principal+reward),
		}, nil
	}
	return nil, fmt.Errorf("stake not found")
}
