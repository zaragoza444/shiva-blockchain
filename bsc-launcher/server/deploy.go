package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type DeployRequest struct {
	Name     string        `json:"name"`
	Symbol   string        `json:"symbol"`
	Decimals int           `json:"decimals"`
	Supply   string        `json:"supply"`
	Chain    string        `json:"chain"`
	Features TokenFeatures `json:"features"`
}

type DeployResult struct {
	ContractAddress string `json:"contractAddress"`
	TxHash          string `json:"txHash"`
	Creator         string `json:"creator"`
	DeployMethod    string `json:"deployMethod"`
}

func parseSupply(supply string, decimals int) (*big.Int, error) {
	supply = strings.TrimSpace(supply)
	if supply == "" {
		return nil, fmt.Errorf("supply required")
	}
	raw := new(big.Int)
	if _, ok := raw.SetString(supply, 10); !ok {
		return nil, fmt.Errorf("invalid supply")
	}
	if raw.Sign() <= 0 {
		return nil, fmt.Errorf("supply must be > 0")
	}
	mult := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	return new(big.Int).Mul(raw, mult), nil
}

func (s *Server) deployContract(ctx context.Context, params InitParams, privateKeyHex string, chain Chain) (*DeployResult, error) {
	keyHex := strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid deployer private key")
	}

	rpcURL := chain.RPCURL
	if rpcURL == "" {
		rpcURL = s.cfg.RPCURL
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("rpc connect: %w", err)
	}
	defer client.Close()

	pub := key.Public().(*ecdsa.PublicKey)
	from := crypto.PubkeyToAddress(*pub)
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas price: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(contractABI()))
	if err != nil {
		return nil, fmt.Errorf("abi parse: %w", err)
	}

	constructorArgs, err := parsedABI.Pack("", params)
	if err != nil {
		return nil, fmt.Errorf("pack constructor: %w", err)
	}

	bytecode := common.FromHex(contractBytecodeHex())
	data := append(bytecode, constructorArgs...)

	chainID := big.NewInt(chain.ChainID)
	tx := types.NewContractCreation(nonce, big.NewInt(0), 6_500_000, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return nil, fmt.Errorf("sign tx: %w", err)
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		return nil, fmt.Errorf("send tx: %w", err)
	}

	receipt, err := waitReceipt(ctx, client, signed.Hash(), 3*time.Minute)
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("deploy tx failed")
	}
	if receipt.ContractAddress == (common.Address{}) {
		return nil, fmt.Errorf("no contract address in receipt")
	}

	return &DeployResult{
		ContractAddress: receipt.ContractAddress.Hex(),
		TxHash:          signed.Hash().Hex(),
		Creator:         from.Hex(),
		DeployMethod:    "backend",
	}, nil
}

func waitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	deadline := time.Now().Add(timeout)
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for tx %s", hash.Hex())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

type verifiedDeploy struct {
	ContractAddress string
	Creator         string
}

func (s *Server) verifyDeployTx(ctx context.Context, txHash, expectedContract, rpcURL string, chainID int64) (*verifiedDeploy, error) {
	if rpcURL == "" {
		rpcURL = s.cfg.RPCURL
	}
	if chainID == 0 {
		chainID = s.cfg.ChainID
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("rpc connect: %w", err)
	}
	defer client.Close()

	hash := common.HexToHash(txHash)
	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("tx not found on chain: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("deploy tx failed on-chain")
	}
	addr := receipt.ContractAddress.Hex()
	if addr == "" || addr == "0x0000000000000000000000000000000000000000" {
		return nil, fmt.Errorf("transaction is not a contract deployment")
	}
	if expectedContract != "" && !strings.EqualFold(addr, expectedContract) {
		// Trust on-chain receipt over client-reported address (checksum / timing issues).
		addr = receipt.ContractAddress.Hex()
	}

	out := &verifiedDeploy{ContractAddress: addr}
	tx, _, err := client.TransactionByHash(ctx, hash)
	if err != nil {
		return out, nil
	}
	signer := types.LatestSignerForChainID(big.NewInt(chainID))
	from, err := types.Sender(signer, tx)
	if err == nil {
		out.Creator = from.Hex()
	}
	return out, nil
}
