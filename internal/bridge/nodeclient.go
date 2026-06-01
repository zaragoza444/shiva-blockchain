package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onex-blockchain/onex/internal/types"
)

type NodeClient struct {
	base string
	hc   *http.Client
}

func NewNodeClient(baseURL string) *NodeClient {
	return &NodeClient{
		base: strings.TrimRight(baseURL, "/"),
		hc:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *NodeClient) Status() (*types.APIStatus, error) {
	resp, err := c.hc.Get(c.base + "/api/v1/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("node status %d", resp.StatusCode)
	}
	var st types.APIStatus
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (c *NodeClient) Balance(addr types.Address) (balance, nonce uint64, err error) {
	resp, err := c.hc.Get(c.base + "/api/v1/balance/" + string(addr))
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Balance uint64 `json:"balance"`
		Nonce   uint64 `json:"nonce"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, 0, err
	}
	return out.Balance, out.Nonce, nil
}

func (c *NodeClient) SubmitTx(tx *types.Transaction) error {
	body, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	resp, err := c.hc.Post(c.base+"/api/v1/tx", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("submit tx: %s", string(b))
	}
	return nil
}

func (c *NodeClient) ProxyRPC(body []byte) ([]byte, int, error) {
	resp, err := c.hc.Post(c.base+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	return out, resp.StatusCode, err
}

func (c *NodeClient) Ping() error {
	_, err := c.Status()
	return err
}
