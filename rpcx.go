package rpcx

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	grpcpool "github.com/processout/grpc-go-pool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	CheckPierTimeout         = 100 * time.Second

	ACCOUNT_KEY = "account"
)

type Interchain struct {
	ID                   string            `json:"id"`
	InterchainCounter    map[string]uint64 `json:"interchain_counter,omitempty"`
	ReceiptCounter       map[string]uint64 `json:"receipt_counter,omitempty"`
	SourceReceiptCounter map[string]uint64 `json:"source_receipt_counter,omitempty"`
}

type Account struct {
	Type          string     `json:"type"`
	Balance       *big.Int   `json:"balance"`
	ContractCount uint64     `json:"contract_count"`
	CodeHash      types.Hash `json:"code_hash"`
}

var _ Client = (*ChainClient)(nil)

type ChainClient struct {
	privateKey crypto.PrivateKey
	logger     Logger
	pool       *ConnectionPool
	ipfsClient *IPFSClient
	//normalSeqNo int64
	//ibtpSeqNo   int64
}

func (cli *ChainClient) GetTransactionByBlockHashAndIndex(blockHash string, index uint64) (*pb.GetTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	client, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := client.conn.Close(); err != nil {
			if err != grpcpool.ErrAlreadyClosed {
				cli.logger.Errorf("close conn err: %s", err)
			}
		}
	}()
	request := &pb.TransactionBlockHashAndIndexMsg{
		BlockHash: blockHash,
		Index:     index,
	}
	msg, err := client.broker.GetTransactionByBlockHashAndIndex(ctx, request)
	return msg, err
}

func (cli *ChainClient) GetTransactionByBlockNumberAndIndex(blockNum uint64, index uint64) (*pb.GetTransactionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	client, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := client.conn.Close(); err != nil {
			if err != grpcpool.ErrAlreadyClosed {
				cli.logger.Errorf("close conn err: %s", err)
			}
		}
	}()
	request := &pb.TransactionBlockNumberAndIndexMsg{
		BlockNumber: blockNum,
		Index:       index,
	}
	msg, err := client.broker.GetTransactionByBlockNumberAndIndex(ctx, request)
	return msg, err
}

// SendRawTransaction send signed transaction
func (cli *ChainClient) SendRawTransaction(tx *pb.BxhTransaction) (string, error) {
	return cli.sendRawTransaction(tx)
}

func (cli *ChainClient) sendRawTransaction(tx *pb.BxhTransaction) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return "", fmt.Errorf("set ctx metadata err: %v", err)
	}

	client, err := cli.pool.getClient()
	if err != nil {
		return "", err
	}
	defer func() {
		if err = client.conn.Close(); err != nil {
			if err != grpcpool.ErrAlreadyClosed {
				cli.logger.Errorf("close conn err: %s", err)
			}
		}
	}()
	msg, err := client.broker.SendTransaction(ctx, tx)
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

// SendRawTransactionWithReceipt send signed transaction with receipt
func (cli *ChainClient) SendRawTransactionWithReceipt(tx *pb.BxhTransaction) (*pb.Receipt, error) {
	txHash, err := cli.sendRawTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("send tx error: %w", err)
	}

	receipt, err := cli.GetReceipt(txHash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (cli *ChainClient) SetCtxMetadata(ctx context.Context) (context.Context, error) {
	addr, err := cli.privateKey.PublicKey().Address()
	if err != nil {
		return nil, fmt.Errorf("get client accout err: %v", err)
	}

	md := metadata.New(map[string]string{ACCOUNT_KEY: addr.String()})
	accountCtx := metadata.NewOutgoingContext(ctx, md)
	return accountCtx, nil
}

func (cli *ChainClient) GetAccountBalance(address string) (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetAccountBalanceTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

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
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
	return response, nil
}

var pool *ConnectionPool

func NewWithNoGlobalPool(opts ...Option) (*ChainClient, error) {
	cfg, err := generateConfig(opts...)
	if err != nil {
		return nil, err
	}

	clientPool, err := NewPool(cfg)
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
		pool:       clientPool,
		ipfsClient: ipfsClient,
	}, nil
}

func New(opts ...Option) (*ChainClient, error) {
	cfg, err := generateConfig(opts...)
	if err != nil {
		return nil, err
	}

	if pool == nil {
		pool, err = NewPool(cfg)
		if err != nil {
			return nil, err
		}
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
	if cli.pool.pool.IsClosed() {
		cli.logger.Warningf("client has been closed")
		return nil
	}
	err := cli.pool.Close()
	if err != nil {
		return err
	}
	cli.pool = nil
	return nil
}

func (cli *ChainClient) SendView(tx *pb.BxhTransaction) (*pb.Receipt, error) {
	return cli.sendView(tx)
}

func (cli *ChainClient) SendTransaction(tx *pb.BxhTransaction, opts *TransactOpts) (string, error) {
	return cli.sendTransaction(tx, opts)
}

func (cli *ChainClient) SendTransactions(txs *pb.MultiTransaction) (*pb.MultiTransactionHash, error) {
	return cli.sendTransactions(txs)
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

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
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

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
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
		opts.PrivKey = cli.privateKey
	}

	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return "", fmt.Errorf("set ctx metadata err: %v", err)
	}
	var nonce uint64
	if opts.Nonce == 0 {
		// no nonce set for tx, then use latest nonce from bitxhub
		nonce, err = cli.GetPendingNonceByAccount(opts.From)
		if err != nil {
			return "", fmt.Errorf("%w: failed to retrieve nonce for account %s for %s", ErrBrokenNetwork, opts.From, err.Error())
		}
	} else {
		nonce = opts.Nonce
	}
	tx.Nonce = nonce

	if opts.PrivKey == nil {
		opts.PrivKey = cli.privateKey
	}
	if err := tx.Sign(opts.PrivKey); err != nil {
		return "", fmt.Errorf("%w: for reason %s", ErrSignTx, err.Error())
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return "", err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			if err != grpcpool.ErrAlreadyClosed {
				cli.logger.Errorf("close conn err: %s", err)
			}
		}
	}()
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

func (cli *ChainClient) sendTransactions(txs *pb.MultiTransaction) (*pb.MultiTransactionHash, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			if err != grpcpool.ErrAlreadyClosed {
				cli.logger.Errorf("close conn err: %s", err)
			}
		}
	}()
	msg, err := grpcClient.broker.SendTransactions(ctx, txs)
	if err != nil {
		st := status.Convert(err)
		switch st.Code() {
		case codes.Unknown, codes.Internal:
			return nil, fmt.Errorf("%w: %s", ErrBrokenNetwork, st.Err().Error())
		case codes.InvalidArgument:
			return nil, fmt.Errorf("%w: %s", ErrReconstruct, st.Err().Error())
		default:
			return nil, err
		}
	}
	return msg, nil
}

func (cli *ChainClient) sendView(tx *pb.BxhTransaction) (*pb.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendTransactionTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
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

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
	response, err := grpcClient.broker.GetReceipt(ctx, &pb.TransactionHashMsg{
		TxHash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBrokenNetwork, err.Error())
	}
	return response, nil
}

func (cli *ChainClient) GetMultiSigns(content string, typ pb.GetSignsRequest_Type) (*pb.SignResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendMultiSignsTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
	response, err := grpcClient.broker.GetMultiSigns(ctx, &pb.GetSignsRequest{
		Content: content,
		Type:    typ,
	})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetTssSigns(content string, typ pb.GetSignsRequest_Type, extra []byte) (*pb.SignResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), SendMultiSignsTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
	response, err := grpcClient.broker.GetTssSigns(ctx, &pb.GetSignsRequest{
		Content: content,
		Type:    typ,
		Extra:   extra,
	})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetPendingNonceByAccount(account string) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return 0, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			if err != grpcpool.ErrAlreadyClosed {
				cli.logger.Errorf("close conn err: %s", err)
			}
		}
	}()
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

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return 0, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
	resp, err := grpcClient.broker.GetTPS(ctx, &pb.GetTPSRequest{
		Begin: begin,
		End:   end,
	})

	if err != nil {
		return 0, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}

	if resp == nil {
		return 0, fmt.Errorf("empty response")
	}

	res := strings.Split(string(resp.Data), " ")
	tps, err := strconv.ParseFloat(res[len(res)-1], 32)
	if err != nil {
		return 0, err
	}
	return uint64(tps), nil
}

func (cli *ChainClient) GetChainID() (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetChainIDTimeout)
	defer cancel()

	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return 0, fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := grpcClient.conn.Close(); err != nil {
			cli.logger.Errorf("close conn err: %s", err)
		}
	}()
	resp, err := grpcClient.broker.GetChainID(ctx, &pb.Empty{})

	if err != nil {
		return 0, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}

	if resp == nil || resp.Data == nil {
		return 0, fmt.Errorf("empty response")
	}

	return binary.LittleEndian.Uint64(resp.Data), nil
}
