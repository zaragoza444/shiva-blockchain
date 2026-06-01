package consensus

import (
	"strings"

	"github.com/onex-blockchain/onex/internal/crypto"
	"github.com/onex-blockchain/onex/internal/types"
)

func MineBlock(b *types.Block) {
	target := strings.Repeat("0", int(b.Header.Difficulty))
	for {
		b.Header.Nonce++
		b.Hash = crypto.Hash(crypto.BlockHeaderPayload(&b.Header))
		if strings.HasPrefix(b.Hash, target) {
			return
		}
	}
}

func ValidProof(b *types.Block) bool {
	if b.Hash == "" {
		return false
	}
	target := strings.Repeat("0", int(b.Header.Difficulty))
	return strings.HasPrefix(b.Hash, target)
}

func HashMatchesHeader(b *types.Block) bool {
	computed := crypto.Hash(crypto.BlockHeaderPayload(&b.Header))
	return computed == b.Hash
}
