package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TokenRecord struct {
	ContractAddress string `json:"contractAddress"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int    `json:"decimals"`
	Supply          string `json:"supply"`
	TxHash          string `json:"txHash"`
	Creator         string `json:"creator"`
	DeployMethod    string `json:"deployMethod"`
	ChainID         int64  `json:"chainId"`
	ChainSlug       string `json:"chainSlug"`
	ChainName       string `json:"chainName"`
	Explorer        string `json:"explorer"`
	CreatedAt       int64  `json:"createdAt"`
}

type TokenStore struct {
	mu   sync.Mutex
	path string
}

func NewTokenStore(dataDir string) *TokenStore {
	return &TokenStore{path: filepath.Join(dataDir, "tokens.json")}
}

func (s *TokenStore) Load() ([]TokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []TokenRecord{}, nil
		}
		return nil, err
	}
	var list []TokenRecord
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *TokenStore) Save(list []TokenRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *TokenStore) Add(rec TokenRecord) error {
	rec.ContractAddress = strings.ToLower(rec.ContractAddress)
	list, err := s.Load()
	if err != nil {
		return err
	}
	for _, t := range list {
		if strings.EqualFold(t.ContractAddress, rec.ContractAddress) {
			return fmt.Errorf("token already registered: %s", rec.ContractAddress)
		}
	}
	if rec.CreatedAt == 0 {
		rec.CreatedAt = time.Now().Unix()
	}
	list = append(list, rec)
	return s.Save(list)
}

func (s *TokenStore) Find(address string) (*TokenRecord, error) {
	list, err := s.Load()
	if err != nil {
		return nil, err
	}
	addr := strings.ToLower(address)
	for i := range list {
		if strings.EqualFold(list[i].ContractAddress, addr) {
			t := list[i]
			return &t, nil
		}
	}
	return nil, fmt.Errorf("token not found")
}
