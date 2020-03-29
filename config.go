package rpcx

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/log"
)

const blockChanNumber = 1024

type config struct {
	addrs      []string
	logger     Logger
	privateKey crypto.PrivateKey
}

type Option func(*config)

func WithAddrs(addrs []string) Option {
	return func(config *config) {
		config.addrs = addrs
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

	if len(config.addrs) == 0 {
		return fmt.Errorf("bitxhub addrs cant not be 0")
	}

	if config.logger == nil {
		config.logger = log.NewWithModule("rpcx")
	}

	return nil
}
