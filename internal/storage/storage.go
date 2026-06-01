package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/onex-blockchain/onex/internal/types"
)

type Store struct {
	dir  string
	mu   sync.RWMutex
	meta chainMeta
}

type chainMeta struct {
	Height uint64 `json:"height"`
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "blocks"), 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir}
	_ = s.loadMeta()
	return s, nil
}

func (s *Store) metaPath() string {
	return filepath.Join(s.dir, "meta.json")
}

func (s *Store) blockPath(index uint64) string {
	return filepath.Join(s.dir, "blocks", fmt.Sprintf("%08d.json", index))
}

func (s *Store) loadMeta() error {
	data, err := os.ReadFile(s.metaPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.meta)
}

func (s *Store) saveMeta() error {
	data, err := json.MarshalIndent(s.meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(), data, 0o644)
}

func (s *Store) Height() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta.Height
}

func (s *Store) PutBlock(b *types.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.blockPath(b.Header.Index), data, 0o644); err != nil {
		return err
	}
	if b.Header.Index > s.meta.Height {
		s.meta.Height = b.Header.Index
	}
	return s.saveMeta()
}

func (s *Store) GetBlock(index uint64) (*types.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.blockPath(index))
	if err != nil {
		return nil, err
	}
	var b types.Block
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) GetTip() (*types.Block, error) {
	h := s.Height()
	if h == 0 {
		return s.GetBlock(0)
	}
	return s.GetBlock(h)
}

func (s *Store) Iterate(from uint64, fn func(*types.Block) error) error {
	for i := from; i <= s.Height(); i++ {
		b, err := s.GetBlock(i)
		if err != nil {
			return err
		}
		if err := fn(b); err != nil {
			return err
		}
	}
	return nil
}
