package ai

import (
	"fmt"
	"strings"
)

// Assistant answers questions about OneX chain and wallet.
type Assistant struct {
	cfg Config
}

func NewAssistant() *Assistant {
	return &Assistant{cfg: LoadConfig()}
}

func (a *Assistant) Status() map[string]interface{} {
	return map[string]interface{}{
		"enabled":   a.cfg.Enabled,
		"cloud":     a.cfg.CloudAvailable(),
		"model":     a.cfg.Model,
		"baseUrl":   a.cfg.BaseURL,
		"localMode": !a.cfg.CloudAvailable(),
	}
}

// Chat uses cloud API when configured, otherwise local rules.
func (a *Assistant) Chat(req ChatRequest) ChatResponse {
	user := lastUserMessage(req.Messages)
	if user == "" {
		return ChatResponse{Reply: "Send a message to OneX AI.", Mode: "local"}
	}

	if a.cfg.CloudAvailable() {
		system := walletSystemHint + "\n\nLive context:\n" + req.Context
		reply, err := cloudChat(a.cfg, system, trimHistory(req.Messages))
		if err == nil && strings.TrimSpace(reply) != "" {
			return ChatResponse{
				Reply: strings.TrimSpace(reply),
				Mode:  "cloud",
				Suggestions: []string{
					"Summarize my portfolio",
					"Best way to stake?",
					"Explain bridge routing",
				},
			}
		}
		// fall through to local on API errors
		local := localReply(user, req.Context)
		if err != nil {
			local.Reply = fmt.Sprintf("Cloud AI unavailable (%v). Local assistant:\n\n%s", err, local.Reply)
		}
		return local
	}

	return localReply(user, req.Context)
}

func lastUserMessage(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func trimHistory(msgs []Message) []Message {
	const max = 20
	if len(msgs) <= max {
		out := make([]Message, 0, len(msgs))
		for _, m := range msgs {
			if m.Role == "user" || m.Role == "assistant" {
				out = append(out, m)
			}
		}
		return out
	}
	return msgs[len(msgs)-max:]
}

// BuildChainContext formats node status for the chain AI endpoint.
func BuildChainContext(chainID string, networkID, height uint64, peers int, mining bool, mempool int) string {
	return fmt.Sprintf("chainId=%s networkId=%d height=%d peers=%d mining=%v mempoolTx=%d",
		chainID, networkID, height, peers, mining, mempool)
}
