package rpcx

import (
	"context"
	"fmt"
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

	response, err := grpcClient.broker.DelVPNode(ctx, &pb.DelVPNodeRequest{Pid: pid})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetValidators() (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	response, err := grpcClient.broker.GetInfo(ctx, &pb.Request{Type: pb.Request_VALIDATORS})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}

func (cli *ChainClient) GetNetworkMeta() (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GetInfoTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	response, err := grpcClient.broker.GetInfo(ctx, &pb.Request{Type: pb.Request_NETWORK})
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
	}
	return response, nil
}
