package ai

import "testing"

func TestLocalReplySwap(t *testing.T) {
	out := localReply("how do I swap tokens?", "")
	if out.Reply == "" {
		t.Fatal("empty reply")
	}
	if out.Mode != "local" {
		t.Fatalf("mode=%s", out.Mode)
	}
	if out.Action == nil || out.Action.Tab != "trade" {
		t.Fatalf("action=%+v", out.Action)
	}
}

func TestChatEmpty(t *testing.T) {
	out := NewAssistant().Chat(ChatRequest{})
	if out.Reply == "" {
		t.Fatal("expected reply")
	}
}
