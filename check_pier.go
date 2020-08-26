package rpcx

import (
	"context"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cli *ChainClient) CheckMasterPier(address string) (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CheckPierTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	request := &pb.Address{
		Address: address,
	}
	return grpcClient.broker.CheckMasterPier(ctx, request)
}

func (cli *ChainClient) SetMasterPier(address string, index string, timeout int64) (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CheckPierTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	request := &pb.PierInfo{
		Address: address,
		Index:   index,
		Timeout: timeout,
	}
	return grpcClient.broker.SetMasterPier(ctx, request)
}

func (cli *ChainClient) HeartBeat(address string, index string) (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CheckPierTimeout)
	defer cancel()

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}
	request := &pb.PierInfo{
		Address: address,
		Index:   index,
	}
	return grpcClient.broker.HeartBeat(ctx, request)
}
