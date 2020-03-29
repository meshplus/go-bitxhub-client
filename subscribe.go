package rpcx

import (
	"context"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/wonderivan/logger"
)

func (cli *ChainClient) Subscribe(ctx context.Context, typ SubscriptionType) (<-chan interface{}, error) {
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	req := &pb.SubscriptionRequest{
		Type: pb.SubscriptionRequest_BLOCK,
	}

	subClient, err := grpcClient.broker.Subscribe(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan interface{})
	go func() {
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := subClient.Recv()
				if err != nil {
					cli.logger.Error("receive: ", err)
					return
				}

				m := &pb.Block{}
				if err := m.Unmarshal(resp.Data); err != nil {
					logger.Error("unmarshal: ", err)
					continue
				}

				c <- m
			}
		}
	}()

	return c, nil
}
