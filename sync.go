package rpcx

import (
	"context"
	"io"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cli *ChainClient) GetBlockHeader(ctx context.Context, begin, end uint64, ch chan<- *pb.BlockHeader) error {
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return err
	}

	syncClient, err := grpcClient.broker.GetBlockHeader(ctx, &pb.GetBlockHeaderRequest{
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

				ch <- resp
			}
		}
	}()

	return nil
}

func (cli *ChainClient) GetInterchainTxWrapper(ctx context.Context, pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrapper) error {
	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return err
	}

	syncClient, err := grpcClient.broker.GetInterchainTxWrapper(ctx, &pb.GetInterchainTxWrapperRequest{
		Begin: begin,
		End:   end,
		Pid: pid,
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

				ch <- resp
			}
		}
	}()

	return nil
}