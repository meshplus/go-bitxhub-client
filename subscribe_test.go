package rpcx

import (
	"context"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
)

func TestChainClient_Subscribe(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	privKey1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	to, err := privKey1.PublicKey().Address()
	require.Nil(t, err)

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := cli.Subscribe(ctx, pb.SubscriptionRequest_BLOCK, nil)
	require.Nil(t, err)

	td := &pb.TransactionData{
		Amount: "10",
	}
	data, err := td.Marshal()
	require.Nil(t, err)

	go func() {
		tx := &pb.BxhTransaction{
			From:      from,
			To:        to,
			Payload:   data,
			Timestamp: time.Now().UnixNano(),
		}

		err = tx.Sign(privKey)
		require.Nil(t, err)

		hash, err := cli.SendTransaction(tx, nil)
		require.Nil(t, err)
		require.EqualValues(t, 66, len(hash))
	}()

	for {
		select {
		case block, ok := <-c:
			require.Equal(t, true, ok)
			require.NotNil(t, block)

			return
		case <-ctx.Done():
			return
		}
	}
}
