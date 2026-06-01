package network

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/chain"
	"github.com/onex-blockchain/onex/internal/legacy"
	"github.com/onex-blockchain/onex/internal/types"
)

const protoVersion = 1

type MessageType string

const (
	MsgHello       MessageType = "hello"
	MsgBlocks      MessageType = "blocks"
	MsgGetBlocks   MessageType = "getblocks"
	MsgTx          MessageType = "tx"
	MsgNewBlock    MessageType = "newblock"
)

type Message struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

type HelloPayload struct {
	Version uint64 `json:"version"`
	Height  uint64 `json:"height"`
	ChainID string `json:"chainId"`
	PeerID  string `json:"peerId"`
}

type GetBlocksPayload struct {
	From uint64 `json:"from"`
}

type Server struct {
	listen   string
	chain    *chain.Blockchain
	chainID  string
	peerID   string
	peers    map[string]*Peer
	mu       sync.RWMutex
	onTx     func(types.Transaction)
	onBlock  func(*types.Block)
}

type Peer struct {
	ID      string
	Addr    string
	Height  uint64
	conn    net.Conn
}

func NewServer(listen string, bc *chain.Blockchain, chainID, peerID string) *Server {
	return &Server{
		listen:  listen,
		chain:   bc,
		chainID: chainID,
		peerID:  peerID,
		peers:   make(map[string]*Peer),
	}
}

func (s *Server) OnTransaction(fn func(types.Transaction)) { s.onTx = fn }
func (s *Server) OnBlock(fn func(*types.Block))           { s.onBlock = fn }

func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.listen)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go s.handleConn(conn)
		}
	}()
	return nil
}

// ConnectWait dials a seed and waits until handshake completes or timeout.
func (s *Server) ConnectWait(ctx context.Context, addr string, timeout time.Duration) error {
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(dctx, "tcp", addr)
	if err != nil {
		return err
	}
	return s.handshake(conn, addr)
}

func (s *Server) Bootstrap(ctx context.Context, seeds []string) {
	for _, seed := range seeds {
		if seed == "" {
			continue
		}
		go func(addr string) {
			if err := s.ConnectWait(ctx, addr, 15*time.Second); err != nil {
				log.Printf("p2p: bootstrap %s: %v", addr, err)
				return
			}
			log.Printf("p2p: connected to seed %s", addr)
			s.syncFromPeers()
		}(seed)
	}
}

func (s *Server) syncFromPeers() {
	height := s.chain.Height()
	from := height + 1
	// request blocks from network - simplified: pull from first peer via getblocks
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.peers {
		if p.Height > height {
			payload, _ := json.Marshal(GetBlocksPayload{From: from})
			msg := Message{Type: MsgGetBlocks, Data: payload}
			_ = writeMsg(p.conn, msg)
		}
	}
}

func (s *Server) handshake(conn net.Conn, addr string) error {
	hello := HelloPayload{
		Version: protoVersion,
		Height:  s.chain.Height(),
		ChainID: s.chainID,
		PeerID:  s.peerID,
	}
	data, _ := json.Marshal(hello)
	if err := writeMsg(conn, Message{Type: MsgHello, Data: data}); err != nil {
		conn.Close()
		return err
	}
	msg, err := readMsg(conn)
	if err != nil {
		conn.Close()
		return err
	}
	if msg.Type != MsgHello {
		conn.Close()
		return fmt.Errorf("expected hello")
	}
	var remote HelloPayload
	if err := json.Unmarshal(msg.Data, &remote); err != nil {
		conn.Close()
		return err
	}
	if remote.ChainID != s.chainID && !chainCompatible(remote.ChainID, s.chainID) {
		conn.Close()
		return fmt.Errorf("chain id mismatch")
	}
	p := &Peer{ID: remote.PeerID, Addr: addr, Height: remote.Height, conn: conn}
	s.mu.Lock()
	s.peers[p.ID] = p
	s.mu.Unlock()
	go s.handleConn(conn)
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		msg, err := readMsg(conn)
		if err != nil {
			return
		}
		switch msg.Type {
		case MsgHello:
			var h HelloPayload
			_ = json.Unmarshal(msg.Data, &h)
			s.mu.Lock()
			s.peers[h.PeerID] = &Peer{ID: h.PeerID, Addr: conn.RemoteAddr().String(), Height: h.Height, conn: conn}
			s.mu.Unlock()
		case MsgTx:
			var tx types.Transaction
			if json.Unmarshal(msg.Data, &tx) == nil && s.onTx != nil {
				s.onTx(tx)
			}
		case MsgGetBlocks:
			var req GetBlocksPayload
			if json.Unmarshal(msg.Data, &req) != nil {
				continue
			}
			var blocks []*types.Block
			for i := req.From; i <= s.chain.Height(); i++ {
				b, err := s.chain.GetBlock(i)
				if err != nil {
					break
				}
				blocks = append(blocks, b)
			}
			data, _ := json.Marshal(blocks)
			_ = writeMsg(conn, Message{Type: MsgBlocks, Data: data})
		case MsgBlocks:
			var blocks []*types.Block
			if json.Unmarshal(msg.Data, &blocks) != nil {
				continue
			}
			s.applyBlocksSequential(blocks)
		case MsgNewBlock:
			var b types.Block
			if json.Unmarshal(msg.Data, &b) == nil {
				s.applyBlocksSequential([]*types.Block{&b})
			}
		}
	}
}

// applyBlocksSequential applies synced blocks in order.
func (s *Server) applyBlocksSequential(blocks []*types.Block) {
	for _, b := range blocks {
		if b == nil {
			continue
		}
		if err := s.chain.ApplyBlock(b); err != nil {
			log.Printf("p2p: apply block %d: %v", b.Header.Index, err)
			return
		}
		if s.onBlock != nil {
			s.onBlock(b)
		}
	}
}

func (s *Server) BroadcastBlock(b *types.Block) {
	data, _ := json.Marshal(b)
	msg := Message{Type: MsgNewBlock, Data: data}
	s.broadcast(msg)
}

func (s *Server) BroadcastTx(tx types.Transaction) {
	data, _ := json.Marshal(tx)
	msg := Message{Type: MsgTx, Data: data}
	s.broadcast(msg)
}

func (s *Server) broadcast(msg Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.peers {
		_ = writeMsg(p.conn, msg)
	}
}

func (s *Server) Peers() []types.PeerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]types.PeerInfo, 0, len(s.peers))
	for _, p := range s.peers {
		out = append(out, types.PeerInfo{ID: p.ID, Address: p.Addr, Height: p.Height})
	}
	return out
}

func (s *Server) PeerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.peers)
}

func readMsg(r io.Reader) (Message, error) {
	br := bufio.NewReader(r)
	line, err := br.ReadBytes('\n')
	if err != nil {
		return Message{}, err
	}
	var m Message
	return m, json.Unmarshal(line, &m)
}

func writeMsg(w io.Writer, m Message) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func chainCompatible(a, b string) bool {
	return legacy.NormalizeChainID(a) == legacy.NormalizeChainID(b)
}
