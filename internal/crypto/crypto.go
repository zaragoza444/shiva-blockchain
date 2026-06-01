package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/onex-blockchain/onex/internal/types"
)

func Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HashConcat(parts ...[]byte) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateKeyPair() (pub types.Address, priv []byte, err error) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, err
	}
	pub = types.Address(hex.EncodeToString(privKey.Public().(ed25519.PublicKey)))
	return pub, privKey, nil
}

func PubFromPrivate(priv []byte) (types.Address, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key length")
	}
	pub := ed25519.PrivateKey(priv).Public().(ed25519.PublicKey)
	return types.Address(hex.EncodeToString(pub)), nil
}

func SignTx(priv []byte, payload []byte) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key")
	}
	sig := ed25519.Sign(ed25519.PrivateKey(priv), payload)
	return hex.EncodeToString(sig), nil
}

func VerifyTx(pub types.Address, payload []byte, sigHex string) bool {
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	pk, err := hex.DecodeString(string(pub))
	if err != nil || len(pk) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pk), payload, sig)
}

func TxPayload(tx *types.Transaction) []byte {
	return []byte(fmt.Sprintf("%s:%s:%d:%d:%d", tx.From, tx.To, tx.Amount, tx.Fee, tx.Nonce))
}

func BlockHeaderPayload(h *types.BlockHeader) []byte {
	return []byte(fmt.Sprintf("%d:%d:%s:%s:%d:%s:%d",
		h.Index, h.Timestamp, h.PreviousHash, h.StateRoot, h.Difficulty, h.Miner, h.Nonce))
}
