package ai

// Message is one chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Action is an optional UI hint for Shiva Wallet.
type Action struct {
	Type string `json:"type"`
	Tab  string `json:"tab,omitempty"`
	Sheet string `json:"sheet,omitempty"`
}

// ChatRequest from clients.
type ChatRequest struct {
	Messages []Message `json:"messages"`
	Context  string    `json:"context,omitempty"`
}

// ChatResponse to clients.
type ChatResponse struct {
	Reply    string   `json:"reply"`
	Mode     string   `json:"mode"`
	Action   *Action  `json:"action,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}
