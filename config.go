package rpcx

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/fileutil"
	"github.com/meshplus/bitxhub-kit/log"
)

const (
	blockChanNumber = 1024
	defaultTimeout  = 1 * time.Second
)

type config struct {
	logger       Logger
	privateKey   crypto.PrivateKey
	nodesInfo    []*NodeInfo
	ipfsAddrs    []string
	timeoutLimit time.Duration // timeout limit config for dialing grpc
}

type NodeInfo struct {
	Addr       string
	EnableTLS  bool
	CertPath   string
	CommonName string
}

type Option func(*config)

func WithNodesInfo(nodesInfo ...*NodeInfo) Option {
	return func(config *config) {
		config.nodesInfo = nodesInfo
	}
}

func WithLogger(logger Logger) Option {
	return func(config *config) {
		config.logger = logger
	}
}

func WithPrivateKey(key crypto.PrivateKey) Option {
	return func(config *config) {
		config.privateKey = key
	}
}

func WithIPFSInfo(addrs []string) Option {
	return func(config *config) {
		config.ipfsAddrs = addrs
	}
}

func WithTimeoutLimit(limit time.Duration) Option {
	return func(config *config) {
		config.timeoutLimit = limit
	}
}

func generateConfig(opts ...Option) (*config, error) {
	config := &config{}
	for _, opt := range opts {
		opt(config)
	}

	if err := checkConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

func checkConfig(config *config) error {
	if config.privateKey == nil {
		return fmt.Errorf("private key is empty")
	}

	if len(config.nodesInfo) == 0 {
		return fmt.Errorf("bitxhub addrs cant not be 0")
	}

	if config.logger == nil {
		config.logger = log.NewWithModule("rpcx")
	}

	if config.timeoutLimit == 0 {
		config.timeoutLimit = defaultTimeout
	}

	// if EnableTLS is set, then tls certs must be provided
	for _, nodeInfo := range config.nodesInfo {
		if nodeInfo.EnableTLS {
			if !fileutil.Exist(filepath.Join(nodeInfo.CertPath, "ca.cert")) {
				return fmt.Errorf("ca.cert file is not found while tls is enabled")
			}
		}
	}
	return nil
}
