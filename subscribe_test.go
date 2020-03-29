package rpcx

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainClient_Subscribe(t *testing.T) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	privKey1, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	to, err := privKey1.PublicKey().Address()
	require.Nil(t, err)

	cli, err := New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := cli.Subscribe(ctx, SubscribeNewBlock)
	assert.Nil(t, err)

	go func() {
		tx := &pb.Transaction{
			From: from,
			To:   to,
			Data: &pb.TransactionData{
				Amount: 10,
			},
			Timestamp: time.Now().UnixNano(),
			Nonce:     rand.Int63(),
		}

		err = tx.Sign(privKey)
		require.Nil(t, err)

		hash, err := cli.SendTransaction(tx)
		require.Nil(t, err)
		require.EqualValues(t, 66, len(hash))
	}()

	for {
		select {
		case block := <-c:
			if block == nil {
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
