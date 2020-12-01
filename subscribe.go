package rpcx

import (
	"context"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cli *ChainClient) Subscribe(ctx context.Context, typ pb.SubscriptionRequest_Type, extra []byte) (<-chan interface{}, error) {
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	req := &pb.SubscriptionRequest{
		Type:  typ,
		Extra: extra,
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

				var ret interface{}
				switch typ {
				case pb.SubscriptionRequest_BLOCK_HEADER:
					header := &pb.BlockHeader{}
					if err := header.Unmarshal(resp.Data); err != nil {
						cli.logger.Error("receive header error: ", resp.Data)
						return
					}
					ret = header
				case pb.SubscriptionRequest_BLOCK:
					block := &pb.Block{}
					if err := block.Unmarshal(resp.Data); err != nil {
						cli.logger.Error("receive header error: ", resp.Data)
						return
					}
					ret = block
				case pb.SubscriptionRequest_EVENT:
					event := &pb.Event{}
					if err := event.Unmarshal(resp.Data); err != nil {
						cli.logger.Error("receive event error: ", resp.Data)
						return
					}
					ret = event
				case pb.SubscriptionRequest_INTERCHAIN_TX:
					ibtp := &pb.IBTP{}
					if err := ibtp.Unmarshal(resp.Data); err != nil {
						cli.logger.Error("receive interchain tx error: ", resp.Data)
						return
					}
					ret = ibtp
				case pb.SubscriptionRequest_INTERCHAIN_TX_WRAPPER:
					wrapper := &pb.InterchainTxWrappers{}
					if err := wrapper.Unmarshal(resp.Data); err != nil {
						cli.logger.Error("receive interchain tx wrapper error: ", resp.Data)
						return
					}
					ret = wrapper
				case pb.SubscriptionRequest_UNION_INTERCHAIN_TX_WRAPPER:
					wrapper := &pb.InterchainTxWrappers{}
					if err := wrapper.Unmarshal(resp.Data); err != nil {
						cli.logger.Error("receive interchain tx wrapper error: ", resp.Data)
						return
					}
					ret = wrapper
				default:
					ret = resp.Data
				}

				c <- ret
			}
		}
	}()

	return c, nil
}
