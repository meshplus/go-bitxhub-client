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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	GetTransactionTimeout    = 10 * time.Second
	SendTransactionTimeout   = 10 * time.Second
	SendMultiSignsTimeout    = 10 * time.Second
	GetReceiptTimeout        = 2 * time.Second
	GetAccountBalanceTimeout = 2 * time.Second
	GetTPSTimeout            = 2 * time.Second
	GetChainIDTimeout        = 2 * time.Second
	CheckPierTimeout         = 60 * time.Second
)

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
	defer cli.release(grpcClient)

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

func (cli *ChainClient) SendView(tx *pb.BxhTransaction) (*pb.Receipt, error) {
	return cli.sendView(tx)
}

func (cli *ChainClient) SendTransaction(tx *pb.BxhTransaction, opts *TransactOpts) (string, error) {
	return cli.sendTransaction(tx, opts)
}

func (cli *ChainClient) SendTransactionWithReceipt(tx *pb.BxhTransaction, opts *TransactOpts) (*pb.Receipt, error) {
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
	defer cli.release(grpcClient)
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
	defer cli.release(grpcClient)
	response, err := grpcClient.broker.GetChainMeta(ctx, &pb.Request{})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) sendTransactionWithReceipt(tx *pb.BxhTransaction, opts *TransactOpts) (*pb.Receipt, error) {
	hash, err := cli.sendTransaction(tx, opts)
	if err != nil {
		return nil, fmt.Errorf("send tx error: %w", err)
	}

	receipt, err := cli.GetReceipt(hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (cli *ChainClient) sendTransaction(tx *pb.BxhTransaction, opts *TransactOpts) (string, error) {
	if tx.From == nil {
		return "", fmt.Errorf("%w: from address can't be empty", ErrReconstruct)
	}
	if opts == nil {
		opts = new(TransactOpts)
		opts.From = tx.From.String() // set default from for opts
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return "", fmt.Errorf("get pool client err: %s", err)
	}
	defer cli.release(grpcClient)
	var nonce uint64
	if opts.Nonce == 0 {
		cli.release(grpcClient)
		// no nonce set for tx, then use latest nonce from bitxhub
		nonce, err = cli.GetPendingNonceByAccount(opts.From)
		if err != nil {
			return "", fmt.Errorf("%w: failed to retrieve nonce for account %s for %s", ErrBrokenNetwork, opts.From, err.Error())
		}
		grpcClient, err = cli.pool.getClient()
		if err != nil {
			return "", fmt.Errorf("get pool client err: %s", err)
		}
	} else {
		nonce = opts.Nonce
	}
	tx.Nonce = nonce

	if err := tx.Sign(cli.privateKey); err != nil {
		return "", fmt.Errorf("%w: for reason %s", ErrSignTx, err.Error())
	}

	msg, err := grpcClient.broker.SendTransaction(ctx, tx)
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.Unknown, codes.Internal:
			return "", fmt.Errorf("%w: %s", ErrBrokenNetwork, st.Err().Error())
		case codes.InvalidArgument:
			return "", fmt.Errorf("%w: %s", ErrReconstruct, st.Err().Error())
		default:
			return "", err
		}
	}
	return msg.TxHash, err
}

func (cli *ChainClient) sendView(tx *pb.BxhTransaction) (*pb.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer cli.release(grpcClient)
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
	defer cli.release(grpcClient)
	response, err := grpcClient.broker.GetReceipt(ctx, &pb.TransactionHashMsg{
		TxHash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBrokenNetwork, err.Error())
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
	defer cli.release(grpcClient)
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
	defer cli.release(grpcClient)
	res, err := grpcClient.broker.GetPendingNonceByAccount(ctx, &pb.Address{
		Address: account,
	})
	if err != nil {
		return 0, err
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
	defer cli.release(grpcClient)
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

func (cli *ChainClient) GetChainID() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetChainIDTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return 0, err
	}
	defer cli.release(grpcClient)
	resp, err := grpcClient.broker.GetChainID(ctx, &pb.Empty{})

	if err != nil {
		return 0, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}

	if resp == nil || resp.Data == nil {
		return 0, fmt.Errorf("empty response")
	}

	return binary.LittleEndian.Uint64(resp.Data), nil
}
