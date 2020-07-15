package rpcx

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var (
	cfg = &config{
		addrs: []string{
			"localhost:60011",
		},
		logger: logrus.New(),
	}
	BoltContractAddress = "0x000000000000000000000000000000000000000b"
	value               = "value"
)

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

func TestChainClient_SendView(t *testing.T) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)

	cli, err := New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	rand.Seed(time.Now().UnixNano())
	randKey := make([]byte, 20)
	_, err = rand.Read(randKey)
	require.Nil(t, err)

	tx, err := genContractTransaction(pb.TransactionData_BVM, privKey,
		types.String2Address(BoltContractAddress), "Set", pb.String(string(randKey)), pb.String(value))
	require.Nil(t, err)

	err = tx.Sign(privKey)
	require.Nil(t, err)

	// test sending write-ledger tx to SendView api
	// bitxhub will execute this tx, but its result will not be persisted in storage
	receipt, err := cli.sendView(tx)
	require.Nil(t, err)
	require.Equal(t, pb.Receipt_SUCCESS, receipt.Status)

	queryKey, err := genContractTransaction(pb.TransactionData_BVM, privKey,
		types.String2Address(BoltContractAddress), "Get", pb.String(string(randKey)))
	require.Nil(t, err)

	receipt, err = cli.SendView(queryKey)
	require.Nil(t, err)
	require.Equal(t, pb.Receipt_FAILED, receipt.Status)
	require.NotEqual(t, value, string(receipt.Ret))

	// test sending write-ledger tx to SendTransaction api
	hash, err := cli.SendTransaction(tx)
	require.Nil(t, err)

	ret, err := cli.GetReceipt(hash)
	require.Nil(t, err)
	require.Equal(t, tx.Hash().String(), ret.TxHash.String())

	// test sending read-ledger tx to SendView api
	view, err := genContractTransaction(pb.TransactionData_BVM, privKey,
		types.String2Address(BoltContractAddress), "Get", pb.String(string(randKey)))
	require.Nil(t, err)

	receipt, err = cli.SendView(view)
	require.Nil(t, err)
	require.Equal(t, value, string(receipt.Ret))

	err = cli.Stop()
	require.Nil(t, err)
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

func genContractTransaction(
	vmType pb.TransactionData_VMType, privateKey crypto.PrivateKey,
	address types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	from, err := privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	pl := &pb.InvokePayload{
		Method: method,
		Args:   args[:],
	}

	data, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  vmType,
		Payload: data,
	}

	tx := &pb.Transaction{
		From:      from,
		To:        address,
		Data:      td,
		Timestamp: time.Now().UnixNano(),
		Nonce:     rand.Int63(),
	}

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	tx.TransactionHash = tx.Hash()

	return tx, nil
}
