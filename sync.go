package rpcx

import (
	"context"
	"io"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/wonderivan/logger"
)

func (cli *ChainClient) SyncMerkleWrapper(ctx context.Context, id string, num uint64) (chan *pb.MerkleWrapper, error) {
	c := make(chan *pb.MerkleWrapper, blockChanNumber)

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return nil, err
	}

	syncClient, err := grpcClient.broker.SyncMerkleWrapper(ctx, &pb.SyncMerkleWrapperRequest{
		AppchainId: id,
	})
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(c)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := syncClient.Recv()
				if err != nil {
					cli.logger.Error(err)
					return
				}

				m := &pb.MerkleWrapper{}
				err = m.Unmarshal(resp.Data)
				if err != nil {
					logger.Error(err)
					continue
				}

				c <- m
			}
		}
	}()

	return c, nil
}

func (cli *ChainClient) GetMerkleWrapper(ctx context.Context, pid string, begin, end uint64, ch chan<- *pb.MerkleWrapper) error {
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return err
	}

	syncClient, err := grpcClient.broker.GetMerkleWrapper(ctx, &pb.GetMerkleWrapperRequest{
		Pid:   pid,
		Begin: begin,
		End:   end,
	})
	if err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := syncClient.Recv()
				if err != nil {
					if err != io.EOF {
						cli.logger.Error(err)
					}
					return
				}

				m := &pb.MerkleWrapper{}
				err = m.Unmarshal(resp.Data)
				if err != nil {
					return
				}

				ch <- m
			}
		}
	}()

	return nil
}
