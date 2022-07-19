package rpcx

import (
	"context"
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cli *ChainClient) SubscribeAudit(ctx context.Context, typ pb.AuditSubscriptionRequest_Type, blockHeight uint64, extra []byte) (<-chan interface{}, error) {
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
	from, err := cli.privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}
	req := &pb.AuditSubscriptionRequest{
		Type:        typ,
		AuditNodeId: from.String(),
		BlockHeight: blockHeight,
		Extra:       extra,
	}

	subClient, err := grpcClient.broker.SubscribeAuditInfo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s, %w", err.Error(), ErrBrokenNetwork)
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
				case pb.AuditSubscriptionRequest_AUDIT_NODE:
					auditInfo := &pb.AuditTxInfo{}
					if err := auditInfo.Unmarshal(resp.Data); err != nil {
						cli.logger.Errorf("receive audit info error: %v", err)
						return
					}
					ret = auditInfo
				default:
					ret = resp.Data
				}

				c <- ret
			}
		}
	}()

	return c, nil
}
