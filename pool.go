package rpcx

import (
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-model/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type grpcClient struct {
	broker pb.ChainBrokerClient
	conn   *grpc.ClientConn
	addr   string
}

type ConnectionPool struct {
	connections []*grpcClient
	logger      Logger
}

// init a connection
func NewPool(config *config) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		logger: config.logger,
	}
	for _, addr := range config.addrs {
		cli := &grpcClient{
			addr: addr,
		}
		pool.connections = append(pool.connections, cli)
	}
	return pool, nil
}

func (pool *ConnectionPool) Close() error {
	for _, c := range pool.connections {
		if c.conn == nil {
			continue
		}
		if err := c.conn.Close(); err != nil {
			pool.logger.Errorf("stop connection with %v error: %v", c.addr, err)
			continue
		}
	}
	return nil
}

// get grpcClient will try to get idle grpcClient
func (pool *ConnectionPool) getClient() (*grpcClient, error) {
	var res *grpcClient
	if err := retry.Retry(func(attempt uint) error {
		for _, cli := range pool.connections {
			if cli.conn == nil || cli.conn.GetState() == connectivity.Shutdown {
				// try to build a connect or reconnect
				conn, err := grpc.Dial(cli.addr, grpc.WithInsecure())
				if err != nil {
					pool.logger.Errorf("dial with addr: %v fail", cli.addr)
					continue
				}
				cli.conn = conn
				cli.broker = pb.NewChainBrokerClient(conn)
				res = cli
				return nil
			}

			s := cli.conn.GetState()
			if s == connectivity.Idle || s == connectivity.Ready {
				res = cli
				return nil
			}
		}
		return fmt.Errorf("all clients are not usable")
	}, strategy.Wait(500*time.Millisecond), strategy.Limit(5)); err != nil {
		return nil, err
	}

	return res, nil
}
