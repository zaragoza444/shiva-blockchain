package ledger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// BankConfig configures bank ledger ingestion.
type BankConfig struct {
	FilePath string
	APIURL   string
	APIKey   string
}

// ReadBankLedger loads fiat balances from a local file or remote banking API.
func ReadBankLedger(cfg BankConfig) ([]Entry, error) {
	var data []byte
	var err error

	switch {
	case cfg.FilePath != "":
		data, err = os.ReadFile(cfg.FilePath)
	case cfg.APIURL != "":
		data, err = fetchBankAPI(cfg.APIURL, cfg.APIKey)
	default:
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseBankJSON(data)
}

func fetchBankAPI(url, apiKey string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("X-API-Key", apiKey)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("bank api status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

func parseBankJSON(data []byte) ([]Entry, error) {
	var file BankFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	var out []Entry
	for i, acct := range file.Accounts {
		cur := strings.ToUpper(strings.TrimSpace(acct.Currency))
		if cur == "" {
			cur = "USD"
		}
		bal := strings.TrimSpace(acct.Balance)
		if bal == "" {
			continue
		}
		id := acct.ID
		if id == "" {
			id = fmt.Sprintf("bank-%d", i)
		}
		account := acct.Name
		if acct.IBAN != "" {
			if account != "" {
				account += " · "
			}
			account += acct.IBAN
		}
		out = append(out, Entry{
			ID:           id,
			Source:       SourceBank,
			Mode:         ModeBank,
			Asset:        cur,
			TokenKey:     "fiat:" + cur,
			Human:        bal,
			FiatCurrency: cur,
			Account:      account,
			Timestamp:    now,
			Reference:    "bank-balance",
		})
	}
	return out, nil
}

// ParseImportLedger normalizes externally supplied ledger rows.
func ParseImportLedger(data []byte) ([]Entry, error) {
	var file ImportFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	var out []Entry
	for i, row := range file.Entries {
		asset := strings.ToUpper(strings.TrimSpace(row.Asset))
		if asset == "" {
			continue
		}
		src := SourceImport
		if row.Source != "" {
			src = SourceKind(strings.ToLower(row.Source))
		}
		mode := ModeReal
		if isFiat(asset) {
			mode = ModeFiat
		}
		cur := strings.ToUpper(strings.TrimSpace(row.Currency))
		if cur == "" {
			if isFiat(asset) {
				cur = asset
			} else {
				cur = "USD"
			}
		}
		out = append(out, Entry{
			ID:           fmt.Sprintf("import-%d", i),
			Source:       src,
			Mode:         mode,
			Asset:        asset,
			TokenKey:     asset,
			Human:        strings.TrimSpace(row.Amount),
			FiatCurrency: cur,
			Account:      row.Account,
			Reference:    row.Reference,
			Timestamp:    now,
		})
	}
	return out, nil
}
