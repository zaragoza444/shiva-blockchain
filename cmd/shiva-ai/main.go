package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/shiva-blockchain/shiva/internal/ai"
)

func main() {
	bridge := flag.String("bridge", "http://127.0.0.1:9338", "bridge base URL (wallet AI)")
	node := flag.String("node", "http://127.0.0.1:8545", "node base URL (chain AI)")
	target := flag.String("target", "wallet", "wallet or chain")
	once := flag.String("ask", "", "single question (non-interactive)")
	flag.Parse()

	base := strings.TrimRight(*bridge, "/")
	chatPath := "/bridge/ai/chat"
	statusPath := "/bridge/ai/status"
	if *target == "chain" {
		base = strings.TrimRight(*node, "/")
		chatPath = "/api/v1/ai/chat"
		statusPath = "/api/v1/ai/status"
	}

	if *once != "" {
		reply, err := postChat(base+chatPath, []ai.Message{{Role: "user", Content: *once}})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(reply)
		return
	}

	st, _ := httpGet(base + statusPath)
	fmt.Println("Shiva AI —", string(st))
	fmt.Println("Type a question (empty line to exit).")

	sc := bufio.NewScanner(os.Stdin)
	var history []ai.Message
	for {
		fmt.Print("you> ")
		if !sc.Scan() {
			break
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			break
		}
		history = append(history, ai.Message{Role: "user", Content: line})
		reply, err := postChat(base+chatPath, history)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println("ai>", reply)
		history = append(history, ai.Message{Role: "assistant", Content: reply})
	}
}

func postChat(url string, msgs []ai.Message) (string, error) {
	body, _ := json.Marshal(ai.ChatRequest{Messages: msgs})
	res, err := httpPost(url, body)
	if err != nil {
		return "", err
	}
	var out ai.ChatResponse
	if err := json.Unmarshal(res, &out); err != nil {
		return "", err
	}
	return out.Reply, nil
}

func httpPost(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return httpDo(req)
}

func httpGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return httpDo(req)
}

func httpDo(req *http.Request) ([]byte, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("%s: %s", res.Status, string(raw))
	}
	return raw, nil
}
