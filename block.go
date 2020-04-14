package rpcx

import (
	"context"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
)

var _ Client = (*ChainClient)(nil)

const (
	GetBlocksTimeout = 10 * time.Second
	GetBlockTimeout  = 10 * time.Second
)

func (cli *ChainClient) GetBlocks(offset uint64, length uint64) (*pb.GetBlocksResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetBlocksTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	request := &pb.GetBlocksRequest{
		Offset: offset,
		Length: length,
	}
	return grpcClient.broker.GetBlocks(ctx, request)
}

func (cli *ChainClient) GetBlock(value string, blockType pb.GetBlockRequest_Type) (*pb.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetBlockTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	request := &pb.GetBlockRequest{
		Type:  blockType,
		Value: value,
	}
	return grpcClient.broker.GetBlock(ctx, request)
}

func (cli *ChainClient) GetChainStatus() (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	return grpcClient.broker.GetInfo(ctx, &pb.Request{Type: pb.Request_CHAIN_STATUS})
}
