package rpcx

import (
	"context"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
)

const (
	GetInfoTimeout   = 2 * time.Second
	DelVPNodeTimeout = 2 * time.Second
)

func (cli *ChainClient) DelVPNode(pid string) (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DelVPNodeTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	return grpcClient.broker.DelVPNode(ctx, &pb.DelVPNodeRequest{pid})
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
