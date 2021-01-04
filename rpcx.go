package rpcx

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-model/pb"
)

const (
	GetTransactionTimeout    = 10 * time.Second
	SendTransactionTimeout   = 10 * time.Second
	SendMultiSignsTimeout    = 10 * time.Second
	GetReceiptTimeout        = 2 * time.Second
	GetAccountBalanceTimeout = 2 * time.Second
	GetTPSTimeout            = 2 * time.Second
	CheckPierTimeout         = 60 * time.Second
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
	privateKey  crypto.PrivateKey
	logger      Logger
	pool        *ConnectionPool
	ipfsClient  *IPFSClient
	normalSeqNo int64
	ibtpSeqNo   int64
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
	response, err := grpcClient.broker.GetAccountBalance(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
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

	ipfsClient, err := NewIPFSClient(WithAPIAddrs(cfg.ipfsAddrs))
	if err != nil {
		return nil, err
	}

	return &ChainClient{
		privateKey: cfg.privateKey,
		logger:     cfg.logger,
		pool:       pool,
		ipfsClient: ipfsClient,
	}, nil
}

func (cli *ChainClient) Stop() error {
	return cli.pool.Close()
}

func (cli *ChainClient) SendView(tx *pb.Transaction) (*pb.Receipt, error) {
	return cli.sendView(tx)
}

func (cli *ChainClient) SendTransaction(tx *pb.Transaction, opts *TransactOpts) (string, error) {
	return cli.sendTransaction(tx, opts)
}

func (cli *ChainClient) SendTransactionWithReceipt(tx *pb.Transaction, opts *TransactOpts) (*pb.Receipt, error) {
	return cli.sendTransactionWithReceipt(tx, opts)
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

	response, err := grpcClient.broker.GetTransaction(ctx, &pb.TransactionHashMsg{
		TxHash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
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

	response, err := grpcClient.broker.GetChainMeta(ctx, &pb.Request{})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) sendTransactionWithReceipt(tx *pb.Transaction, opts *TransactOpts) (*pb.Receipt, error) {
	hash, err := cli.sendTransaction(tx, opts)
	if err != nil {
		return nil, fmt.Errorf("send tx error: %s", err)
	}

	receipt, err := cli.GetReceipt(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (cli *ChainClient) sendTransaction(tx *pb.Transaction, opts *TransactOpts) (string, error) {
	if tx.From == nil {
		return "", fmt.Errorf("from address can't be empty")
	}
	if opts == nil {
		opts = new(TransactOpts)
		opts.From = tx.From.String() // set default from for opts
	}

	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return "", err
	}

	if opts.IBTPNonce != 0 && opts.NormalNonce != 0 {
		return "", fmt.Errorf("can't set ibtp nonce and normal nonce at the same time")
	}

	var nonce uint64
	if opts.IBTPNonce == 0 && opts.NormalNonce == 0 {
		// not nonce set for tx, then use latest nonce from bitxhub
		nonce, err = cli.GetPendingNonceByAccount(opts.From)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve account nonce: %w", err)
		}
	} else {
		if opts.IBTPNonce != 0 {
			nonce = opts.IBTPNonce
		} else {
			nonce = opts.NormalNonce
		}
	}
	tx.Nonce = nonce

	if err := tx.Sign(cli.privateKey); err != nil {
		return "", fmt.Errorf("tx sign: %w", err)
	}

	msg, err := grpcClient.broker.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
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

	if err := tx.Sign(cli.privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	receipt, err := grpcClient.broker.SendView(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
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

	response, err := grpcClient.broker.GetReceipt(ctx, &pb.TransactionHashMsg{
		TxHash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetMultiSigns(content string, typ pb.GetMultiSignsRequest_Type) (*pb.SignResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendMultiSignsTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	response, err := grpcClient.broker.GetMultiSigns(ctx, &pb.GetMultiSignsRequest{
		Content: content,
		Type:    typ,
	})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetPendingNonceByAccount(account string) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return 0, err
	}

	res, err := grpcClient.broker.GetPendingNonceByAccount(ctx, &pb.Address{
		Address: account,
	})
	if err != nil {
		return 0, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return strconv.ParseUint(string(res.Data), 10, 64)
}

func CheckReceipt(receipt *pb.Receipt) bool {
	return receipt.Status == pb.Receipt_SUCCESS
}

func (cli *ChainClient) GetTPS(begin, end uint64) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetTPSTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return 0, err
	}

	resp, err := grpcClient.broker.GetTPS(ctx, &pb.GetTPSRequest{
		Begin: begin,
		End:   end,
	})

	if err != nil {
		return 0, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}

	if resp == nil || resp.Data == nil {
		return 0, fmt.Errorf("empty response")
	}

	return binary.LittleEndian.Uint64(resp.Data), nil
}
