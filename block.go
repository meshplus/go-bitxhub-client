package rpcx

import (
	"context"
	"fmt"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
)

var _ Client = (*ChainClient)(nil)

const (
	GetBlocksTimeout = 10 * time.Second
	GetBlockTimeout  = 10 * time.Second
)

func (cli *ChainClient) GetBlocks(start uint64, end uint64) (*pb.GetBlocksResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetBlocksTimeout)
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
	request := &pb.GetBlocksRequest{
		Start: start,
		End:   end,
	}
	response, err := grpcClient.broker.GetBlocks(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetBlock(value string, blockType pb.GetBlockRequest_Type) (*pb.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetBlockTimeout)
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
	request := &pb.GetBlockRequest{
		Type:  blockType,
		Value: value,
	}
	response, err := grpcClient.broker.GetBlock(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetChainStatus() (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
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
	response, err := grpcClient.broker.GetInfo(ctx, &pb.Request{Type: pb.Request_CHAIN_STATUS})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}
