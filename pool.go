package rpcx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-model/pb"
	grpcpool "github.com/processout/grpc-go-pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

type grpcClient struct {
	broker pb.ChainBrokerClient
	conn   *grpcpool.ClientConn
}

type ConnectionPool struct {
	timeoutLimit  time.Duration // timeout limit config for dialing grpc
	pool          *grpcpool.Pool
	currentClient *grpcClient
	logger        Logger
	config        *config
	clientCnt     uint64
}

// NewPool init a connection
func NewPool(config *config) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		config:       config,
		logger:       config.logger,
		timeoutLimit: config.timeoutLimit,
	}
	grpcPool, err := grpcpool.New(pool.newClient, 4, config.poolSize, 1*time.Hour)
	if err != nil {
		return nil, err
	}

	pool.pool = grpcPool
	return pool, nil
}

func (pool *ConnectionPool) Close() error {
	pool.pool.Close()
	return nil
}

func (pool *ConnectionPool) getClient() (*grpcClient, error) {
	//if pool.currentClient != nil && pool.currentClient.available() {
	//	return pool.currentClient, nil
	//}
	conn, err := pool.pool.Get(context.Background())
	if err != nil {
		return nil, err
	}
	pool.currentClient = &grpcClient{
		broker: pb.NewChainBrokerClient(conn.ClientConn),
		conn:   conn,
	}
	return pool.currentClient, nil
}

// get grpcClient will try to get idle grpcClient
func (pool *ConnectionPool) newClient() (*grpc.ClientConn, error) {
	var (
		conn    *grpc.ClientConn
		tlsErr  error
		connErr error
	)
	if connErr = retry.Retry(func(attempt uint) error {
		randGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomIndex := randGenerator.Intn(len(pool.config.nodesInfo))
		nodeInfo := pool.config.nodesInfo[randomIndex]
		// try to build a connect or reconnect
		opts := []grpc.DialOption{grpc.WithBlock(), grpc.WithTimeout(pool.timeoutLimit)}
		// if EnableTLS is set, then setup connection with ca cert
		if nodeInfo.EnableTLS {
			certPathByte, tlsErr := ioutil.ReadFile(nodeInfo.CertPath)
			if tlsErr != nil {
				return nil
			}
			cp := x509.NewCertPool()
			if !cp.AppendCertsFromPEM(certPathByte) {
				tlsErr = fmt.Errorf("credentials: failed to append certificates")
				return nil
			}
			cert, err := tls.LoadX509KeyPair(nodeInfo.AccessCert, nodeInfo.AccessKey)
			if err != nil {
				pool.logger.Debugf("Creat tls credentials from %s for client %s", nodeInfo.CertPath, nodeInfo.Addr)
				tlsErr = fmt.Errorf("%w: tls config is not right", ErrBrokenNetwork)
				return nil
			}
			creds := credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{cert}, ServerName: nodeInfo.CommonName, RootCAs: cp})

			opts = append(opts, grpc.WithTransportCredentials(creds))
		} else {
			opts = append(opts, grpc.WithInsecure())
		}
		conn, connErr = grpc.Dial(nodeInfo.Addr, opts...)
		if connErr != nil {
			pool.logger.Infof("Dial with addr: %s fail", nodeInfo.Addr)
			return fmt.Errorf("%w: dial node %s failed", ErrBrokenNetwork, nodeInfo.Addr)
		}
		pool.logger.Debugf("Establish connection with bitxhub %s successfully, pool is %d pool conn cnt is %d", nodeInfo.Addr, pool.pool.Available(), atomic.AddUint64(&pool.clientCnt, 1))
		return nil
	}, strategy.Wait(500*time.Millisecond), strategy.Limit(uint(5*len(pool.config.nodesInfo)))); connErr != nil {
		return nil, connErr
	}

	if tlsErr != nil {
		return nil, tlsErr
	}

	return conn, nil
}

func (grpcCli *grpcClient) available() bool {
	if grpcCli.conn.ClientConn == nil {
		return false
	}
	s := grpcCli.conn.ClientConn.GetState()
	return s == connectivity.Idle || s == connectivity.Ready
}
