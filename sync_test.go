package rpcx

import (
	"context"
	"fmt"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"math/rand"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainClient_GetBlockHeader(t *testing.T) {
	cli, privKey, from, to := prepareKeypair(t)

	go send(t, cli, from, to, privKey)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan *pb.BlockHeader)
	assert.Nil(t, cli.GetBlockHeader(ctx, 1, 2, ch))

	for {
		select {
		case header := <-ch:
			if header == nil {
				assert.Error(t, fmt.Errorf("channel is closed"))
				return
			}
			if err := cli.Stop(); err != nil {
				return
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func TestChainClient_GetInterchainTxWrapper(t *testing.T) {
	cli, privKey, from, to := prepareKeypair(t)

	go send(t, cli, from, to, privKey)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan *pb.InterchainTxWrapper)
	assert.Nil(t, cli.GetInterchainTxWrapper(ctx,to.String(), 1, 2, ch))

	for {
		select {
		case header := <-ch:
			if header == nil {
				assert.Error(t, fmt.Errorf("channel is closed"))
				return
			}
			if err := cli.Stop(); err != nil {
				return
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func prepareKeypair(t *testing.T) (cli *ChainClient, privKey crypto.PrivateKey, from, to types.Address){
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)
	privKey1, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	cli, err = New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	from, err = privKey.PublicKey().Address()
	require.Nil(t, err)

	to, err = privKey1.PublicKey().Address()
	require.Nil(t, err)

	return cli, privKey, from, to
}

func send(t *testing.T, cli *ChainClient, from, to types.Address, privKey crypto.PrivateKey) {
	tx := &pb.Transaction{
		From: from,
		To:   to,
		Data: &pb.TransactionData{
			Amount: 10,
		},
		Timestamp: time.Now().UnixNano(),
		Nonce:     rand.Int63(),
	}

	err := tx.Sign(privKey)
	require.Nil(t, err)

	hash, err := cli.SendTransaction(tx)
	require.Nil(t, err)
	require.EqualValues(t, 66, len(hash))
}