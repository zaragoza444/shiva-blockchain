package tokenplatform

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func SanitizeTokenID(symbol string) string {
	re := regexp.MustCompile(`[^A-Z0-9]`)
	s := strings.ToUpper(symbol)
	s = re.ReplaceAllString(s, "")
	if len(s) < 2 {
		s = "TKN" + newID()[:4]
	}
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}

func WrapTokenID(originID, targetChainID string) string {
	base := "W" + originID
	if len(base) > 10 {
		base = base[:10]
	}
	h := sha256.Sum256([]byte("wrap:" + originID + ":" + targetChainID))
	return base + strings.ToUpper(hex.EncodeToString(h[:2]))
}

func newID() string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h[:8])
}

func NowUnix() int64 {
	return time.Now().Unix()
}

func UniqueTokenID(tokens []PlatformToken, chainID, baseID string) string {
	tokenID := baseID
	for _, t := range tokens {
		if t.ChainID == chainID && t.ID == tokenID {
			tokenID = baseID + newID()[:4]
			break
		}
	}
	return tokenID
}
