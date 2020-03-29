package rpcx

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var cfg = &config{
	addrs: []string{
		"localhost:60019",
	},
	logger: logrus.New(),
}

func TestChainClient_SendTransactionWithReceipt(t *testing.T) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	privKey1, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	cli, err := New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	to, err := privKey1.PublicKey().Address()
	require.Nil(t, err)

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

	ret, err := cli.GetReceipt(hash)
	require.Nil(t, err)
	require.Equal(t, tx.Hash().String(), ret.TxHash.String())

	err = cli.Stop()
	require.Nil(t, err)
	fmt.Println(ret.TxHash.Hex())
}

func TestChainClient_GetTransaction(t *testing.T) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	privKey1, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	cli, err := New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	to, err := privKey1.PublicKey().Address()
	require.Nil(t, err)

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

	receipt, err := cli.SendTransactionWithReceipt(tx)
	require.Nil(t, err)
	require.True(t, strings.Contains(string(receipt.GetRet()), "not sufficient funds"))

	txx, err := cli.GetTransaction(receipt.TxHash.Hex())
	require.Nil(t, err)
	require.Equal(t, tx.SignHash(), txx.Tx.SignHash())
}

func TestChainClient_GetChainMeta(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	chainMeta, err := cli.GetChainMeta()
	require.Nil(t, err)
	require.True(t, chainMeta.GetHeight() > 0)
}

func TestChainClient_GetNetworkMeta(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	res, err := cli.GetNetworkMeta()
	require.Nil(t, err)
	require.NotNil(t, res)
}

func TestChainClient_GetAccountBalance(t *testing.T) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	cli, err := New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)
	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)
	res, err := cli.GetAccountBalance(address.String())
	require.Nil(t, err)
	require.NotNil(t, res)
}
