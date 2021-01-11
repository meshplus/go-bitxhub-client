package rpcx

import (
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-model/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

type grpcClient struct {
	broker   pb.ChainBrokerClient
	conn     *grpc.ClientConn
	nodeInfo *NodeInfo
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
	for _, nodeInfo := range config.nodesInfo {
		cli := &grpcClient{
			nodeInfo: nodeInfo,
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
			pool.logger.Errorf("stop connection with %v error: %v", c.nodeInfo.Addr, err)
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
				var (
					opt grpc.DialOption
				)
				// if EnableTLS is set, then setup connection with ca cert
				if cli.nodeInfo.EnableTLS {
					creds, err := credentials.NewClientTLSFromFile(cli.nodeInfo.CertPath, cli.nodeInfo.CommonName)
					if err != nil {
						pool.logger.Errorf("creat tls credentials from %s", cli.nodeInfo.CertPath)
						continue
					}
					opt = grpc.WithTransportCredentials(creds)
				} else {
					opt = grpc.WithInsecure()
				}
				conn, err := grpc.Dial(cli.nodeInfo.Addr, opt)
				if err != nil {
					pool.logger.Errorf("dial with Addr: %v fail", cli.nodeInfo.Addr)
					continue
				}
				cli.conn = conn
				cli.broker = pb.NewChainBrokerClient(conn)
				res = cli
				return nil
			}

			s := cli.conn.GetState()
			if s == connectivity.Idle || s == connectivity.Ready || s == connectivity.Connecting {
				res = cli
				return nil
			}
		}
		return newBrokenNetworkError(fmt.Errorf("all clients are not usable"))
	}, strategy.Wait(500*time.Millisecond), strategy.Limit(5)); err != nil {
		return nil, err
	}

	return res, nil
}
