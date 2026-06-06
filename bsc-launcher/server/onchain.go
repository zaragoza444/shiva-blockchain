package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type OnChainTokenInfo struct {
	ContractAddress string `json:"contractAddress"`
	TokenName       string `json:"tokenName"`
	Symbol          string `json:"symbol"`
	Divisor         string `json:"divisor"`
	TotalSupply     string `json:"totalSupply"`
	IsContract      bool   `json:"isContract"`
}

func (s *Server) rpcClient(ctx context.Context, rpcURL string) (*ethclient.Client, error) {
	if rpcURL == "" {
		rpcURL = s.cfg.RPCURL
	}
	return ethclient.DialContext(ctx, rpcURL)
}

func (s *Server) isContractOn(ctx context.Context, rpcURL, address string) (bool, error) {
	client, err := s.rpcClient(ctx, rpcURL)
	if err != nil {
		return false, err
	}
	defer client.Close()

	addr := common.HexToAddress(address)
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, err
	}
	return len(code) > 0, nil
}

func (s *Server) isContract(ctx context.Context, address string) (bool, error) {
	return s.isContractOn(ctx, s.cfg.RPCURL, address)
}

func (s *Server) readOnChainTokenOn(ctx context.Context, chain Chain, address string) (*OnChainTokenInfo, error) {
	rpcURL := chain.RPCURL
	if rpcURL == "" {
		rpcURL = s.cfg.RPCURL
	}
	isContract, err := s.isContractOn(ctx, rpcURL, address)
	if err != nil {
		return nil, err
	}
	info := &OnChainTokenInfo{
		ContractAddress: common.HexToAddress(address).Hex(),
		IsContract:      isContract,
	}
	if !isContract {
		return info, fmt.Errorf("%s is a wallet address, not a token contract", address)
	}

	client, err := s.rpcClient(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	parsedABI, err := abi.JSON(strings.NewReader(contractABI()))
	if err != nil {
		return nil, err
	}
	addr := common.HexToAddress(address)

	if name, err := callString(ctx, client, parsedABI, addr, "name"); err == nil {
		info.TokenName = name
	}
	if symbol, err := callString(ctx, client, parsedABI, addr, "symbol"); err == nil {
		info.Symbol = symbol
	}
	if decimals, err := callUint8(ctx, client, parsedABI, addr, "decimals"); err == nil {
		info.Divisor = fmt.Sprintf("%d", decimals)
	}
	if supply, err := callBigInt(ctx, client, parsedABI, addr, "totalSupply"); err == nil {
		info.TotalSupply = supply.String()
	}
	return info, nil
}

func (s *Server) readOnChainToken(ctx context.Context, address string) (*OnChainTokenInfo, error) {
	return s.readOnChainTokenOn(ctx, defaultChain(s.cfg), address)
}

func (s *Server) tokenInfoForChain(ctx context.Context, chain Chain, address string) (*BSCScanTokenInfo, error) {
	if chain.ChainID == 0 || chain.RPCURL == "" {
		return &BSCScanTokenInfo{
			ContractAddress: address,
			Error:           fmt.Sprintf("%s tracking uses explorer links — deploy OneX tokens on EVM chains", chain.Name),
		}, nil
	}

	onchain, chainErr := s.readOnChainTokenOn(ctx, chain, address)
	if chainErr != nil {
		if onchain != nil && !onchain.IsContract {
			return &BSCScanTokenInfo{
				ContractAddress: address,
				IsWallet:        true,
				Error:           chainErr.Error(),
			}, chainErr
		}
		return nil, chainErr
	}

	scan, _ := s.bscscan.TokenInfoForChain(chain.ChainID, address)
	return mergeTokenInfo(scan, onchain), nil
}

func callString(ctx context.Context, client *ethclient.Client, parsed abi.ABI, addr common.Address, method string) (string, error) {
	data, err := parsed.Pack(method)
	if err != nil {
		return "", err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: data}, nil)
	if err != nil {
		return "", err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return "", err
	}
	if s, ok := vals[0].(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("unexpected %s return", method)
}

func callUint8(ctx context.Context, client *ethclient.Client, parsed abi.ABI, addr common.Address, method string) (uint8, error) {
	data, err := parsed.Pack(method)
	if err != nil {
		return 0, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: data}, nil)
	if err != nil {
		return 0, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return 0, err
	}
	switch v := vals[0].(type) {
	case uint8:
		return v, nil
	case *big.Int:
		return uint8(v.Uint64()), nil
	default:
		return 0, fmt.Errorf("unexpected %s return", method)
	}
}

func callBigInt(ctx context.Context, client *ethclient.Client, parsed abi.ABI, addr common.Address, method string) (*big.Int, error) {
	data, err := parsed.Pack(method)
	if err != nil {
		return nil, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return nil, err
	}
	if v, ok := vals[0].(*big.Int); ok {
		return v, nil
	}
	return nil, fmt.Errorf("unexpected %s return", method)
}

func mergeTokenInfo(scan *BSCScanTokenInfo, chain *OnChainTokenInfo) *BSCScanTokenInfo {
	out := &BSCScanTokenInfo{ContractAddress: chain.ContractAddress}
	if scan != nil {
		*out = *scan
	}
	if chain == nil {
		return out
	}
	if out.TokenName == "" {
		out.TokenName = chain.TokenName
	}
	if out.Symbol == "" {
		out.Symbol = chain.Symbol
	}
	if out.Divisor == "" {
		out.Divisor = chain.Divisor
	}
	if out.TotalSupply == "" {
		out.TotalSupply = chain.TotalSupply
	}
	if out.ContractAddress == "" {
		out.ContractAddress = chain.ContractAddress
	}
	return out
}
