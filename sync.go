package rpcx

import (
	"context"
	"fmt"
	"io"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cli *ChainClient) GetBlockHeader(ctx context.Context, begin, end uint64, ch chan<- *pb.BlockHeader) error {
	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return err
	}

	syncClient, err := grpcClient.broker.GetBlockHeader(ctx, &pb.GetBlockHeaderRequest{
		Begin: begin,
		End:   end,
	})
	if err != nil {
		return fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
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

func (cli *ChainClient) GetInterchainTxWrappers(ctx context.Context, pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error {
	ctx, err := cli.SetCtxMetadata(ctx)
	if err != nil {
		return fmt.Errorf("set ctx metadata err: %v", err)
	}

	grpcClient, err := cli.pool.getClient()
	if err != nil {
		return err
	}

	syncClient, err := grpcClient.broker.GetInterchainTxWrappers(ctx, &pb.GetInterchainTxWrappersRequest{
		Begin: begin,
		End:   end,
		Pid:   pid,
	})
	if err != nil {
		return fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
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
