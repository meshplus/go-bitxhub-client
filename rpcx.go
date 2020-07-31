package rpcx

import (
	"context"
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

var (
	InterchainContractAddr     = types.String2Address("000000000000000000000000000000000000000a")
	StoreContractAddr          = types.String2Address("000000000000000000000000000000000000000b")
	RuleManagerContractAddr    = types.String2Address("000000000000000000000000000000000000000c")
	RoleContractAddr           = types.String2Address("000000000000000000000000000000000000000d")
	AppchainMgrContractAddr    = types.String2Address("000000000000000000000000000000000000000e")
	TransactionMgrContractAddr = types.String2Address("000000000000000000000000000000000000000f")
	AssetExchangeContractAddr  = types.String2Address("0000000000000000000000000000000000000010")
)

const (
	GetTransactionTimeout    = 10 * time.Second
	SendTransactionTimeout   = 10 * time.Second
	GetReceiptTimeout        = 2 * time.Second
	GetAccountBalanceTimeout = 2 * time.Second
	GetInfoTimeout           = 2 * time.Second
)

type Appchain struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Validators    string `json:"validators"`
	ConsensusType int32  `json:"consensus_type"`
	// 0 => registered, 1 => approved, -1 => rejected
	Status    int32  `json:"status"`
	ChainType string `json:"chain_type"`
	Desc      string `json:"desc"`
	Version   string `json:"version"`
	PublicKey string `json:"public_key"`
}

type Interchain struct {
	ID                   string            `json:"id"`
	InterchainCounter    map[string]uint64 `json:"interchain_counter,omitempty"`
	ReceiptCounter       map[string]uint64 `json:"receipt_counter,omitempty"`
	SourceReceiptCounter map[string]uint64 `json:"source_receipt_counter,omitempty"`
}

var _ Client = (*ChainClient)(nil)

type ChainClient struct {
	privateKey crypto.PrivateKey
	logger     Logger
	pool       *ConnectionPool
}

func (cli *ChainClient) GetValidators() (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	return grpcClient.broker.GetInfo(ctx, &pb.Request{Type: pb.Request_VALIDATORS})
}

func (cli *ChainClient) GetNetworkMeta() (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	return grpcClient.broker.GetInfo(ctx, &pb.Request{Type: pb.Request_NETWORK})
}

func (cli *ChainClient) GetAccountBalance(address string) (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetAccountBalanceTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	request := &pb.Address{
		Address: address,
	}
	return grpcClient.broker.GetAccountBalance(ctx, request)
}

func New(opts ...Option) (*ChainClient, error) {
	cfg, err := generateConfig(opts...)
	if err != nil {
		return nil, err
	}

	pool, err := NewPool(cfg)
	if err != nil {
		return nil, err
	}

	return &ChainClient{
		privateKey: cfg.privateKey,
		logger:     cfg.logger,
		pool:       pool,
	}, nil
}

func (cli *ChainClient) Stop() error {
	return cli.pool.Close()
}

func (cli *ChainClient) SendView(tx *pb.Transaction) (*pb.Receipt, error) {
	return cli.sendView(tx)
}

func (cli *ChainClient) SendTransaction(tx *pb.Transaction) (string, error) {
	return cli.sendTransaction(tx)
}

func (cli *ChainClient) SendTransactionWithReceipt(tx *pb.Transaction) (*pb.Receipt, error) {
	return cli.sendTransactionWithReceipt(tx)
}

// GetReceipts get receipts by tx hashes
func (cli *ChainClient) GetReceipt(hash string) (*pb.Receipt, error) {
	var receipt *pb.Receipt
	var err error

	err = retry.Retry(func(attempt uint) error {
		receipt, err = cli.getReceipt(hash)
		if err != nil {
			return err
		}

		return nil
	},
		strategy.Limit(5),
		strategy.Backoff(backoff.Fibonacci(500*time.Millisecond)),
	)

	if err != nil {
		return nil, err
	}

	return receipt, nil
}

func (cli *ChainClient) GetTransaction(hash string) (*pb.GetTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetTransactionTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	return grpcClient.broker.GetTransaction(ctx, &pb.TransactionHashMsg{
		TxHash: hash,
	})
}

func (cli *ChainClient) SetPrivateKey(key crypto.PrivateKey) {
	cli.privateKey = key
}

func (cli *ChainClient) GetChainMeta() (*pb.ChainMeta, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	return grpcClient.broker.GetChainMeta(ctx, &pb.Request{})
}

func (cli *ChainClient) sendTransactionWithReceipt(tx *pb.Transaction) (*pb.Receipt, error) {
	hash, err := cli.sendTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("send tx error: %s", err)
	}

	receipt, err := cli.GetReceipt(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (cli *ChainClient) sendTransaction(tx *pb.Transaction) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return "", err
	}

	req := &pb.SendTransactionRequest{
		Version:   tx.Version,
		From:      tx.From,
		To:        tx.To,
		Timestamp: tx.Timestamp,
		Data:      tx.Data,
		Nonce:     tx.Nonce,
		Signature: tx.Signature,
		Extra:     tx.Extra,
	}

	msg, err := grpcClient.broker.SendTransaction(ctx, req)
	if err != nil {
		return "", err
	}

	return msg.TxHash, err
}

func (cli *ChainClient) sendView(tx *pb.Transaction) (*pb.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	req := &pb.SendTransactionRequest{
		Version:   tx.Version,
		From:      tx.From,
		To:        tx.To,
		Timestamp: tx.Timestamp,
		Data:      tx.Data,
		Nonce:     tx.Nonce,
		Signature: tx.Signature,
		Extra:     tx.Extra,
	}

	receipt, err := grpcClient.broker.SendView(ctx, req)
	if err != nil {
		return nil, err
	}

	return receipt, nil
}

func (cli *ChainClient) getReceipt(hash string) (*pb.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetReceiptTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	return grpcClient.broker.GetReceipt(ctx, &pb.TransactionHashMsg{
		TxHash: hash,
	})
}

func (cli *ChainClient) GetAssetExchangeSigns(id string) (*pb.SignResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetReceiptTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	return grpcClient.broker.GetAssetExchangeSigns(ctx, &pb.AssetExchangeSignsRequest{
		Id: id,
	})
}

func CheckReceipt(receipt *pb.Receipt) bool {
	return receipt.Status == pb.Receipt_SUCCESS
}
