package rpcx

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	ErrBalance = "insufficient balance"
	keyFile    = "key.json"
)

var (
	cfg = &config{
		nodesInfo: []*NodeInfo{
			{Addr: "localhost:60011", EnableTLS: false, CertPath: "testdata/node1/certs/agency.cert", CommonName: "BitXHub",
				AccessCert: "testdata/node1/certs/gateway.cert", AccessKey: "testdata/node1/certs/gateway.priv"},
			//{Addr: "localhost:60012", EnableTLS: true, CertPath: "testdata/node1/certs/agency.cert", CommonName: "BitXHub",
			//	AccessCert: "testdata/node2/certs/gateway.cert", AccessKey: "testdata/node2/certs/gateway.priv"},
			//{Addr: "localhost:60013", EnableTLS: true, CertPath: "testdata/node3/certs/agency.cert", CommonName: "BitXHub",
			//	AccessCert: "testdata/node3/certs/gateway.cert", AccessKey: "testdata/node3/certs/gateway.priv"},
			//{Addr: "localhost:60014", EnableTLS: true, CertPath: "testdata/node4/certs/agency.cert", CommonName: "BitXHub",
			//	AccessCert: "testdata/node4/certs/gateway.cert", AccessKey: "testdata/node4/certs/gateway.priv"},
		},
		logger: logrus.New(),
	}
	cfg1 = &config{
		nodesInfo: []*NodeInfo{
			{Addr: "localhost:60012"},
		},
		logger: logrus.New(),
	}
	BoltContractAddress = "0x000000000000000000000000000000000000000b"
	value               = "value"
)

func TestChainClient_SendTransactionWithReceipt(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	privKey1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	to, err := privKey1.PublicKey().Address()
	require.Nil(t, err)

	data := &pb.TransactionData{
		Amount: "10",
	}

	payload, err := data.Marshal()
	require.Nil(t, err)
	tx := &pb.BxhTransaction{
		From:      from,
		To:        to,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
	}

	err = tx.Sign(privKey)
	require.Nil(t, err)

	hash, err := cli.SendTransaction(tx, nil)
	require.Nil(t, err)

	ret, err := cli.GetReceipt(hash)
	require.Nil(t, err)

	require.Equal(t, hash, ret.TxHash.String())
}

func TestChainClient_SendView(t *testing.T) {
	privKey, err := asym.RestorePrivateKey(filepath.Join("testdata", "key.json"), "bitxhub")
	require.Nil(t, err)

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	rand.Seed(time.Now().UnixNano())
	randKey := make([]byte, 20)
	_, err = rand.Read(randKey)
	require.Nil(t, err)

	tx, err := genContractTransaction(pb.TransactionData_BVM, privKey,
		types.NewAddressByStr(BoltContractAddress), "Set", pb.String(string(randKey)), pb.String(value))
	require.Nil(t, err)
	tx.Nonce = rand.Uint64()

	err = tx.Sign(privKey)
	require.Nil(t, err)

	// test sending write-ledger tx to SendView api
	// bitxhub will execute this tx, but its result will not be persisted in storage
	receipt, err := cli.sendView(tx)
	require.Nil(t, err)
	require.Equal(t, pb.Receipt_SUCCESS, receipt.Status, string(receipt.Ret))

	queryKey, err := genContractTransaction(pb.TransactionData_BVM, privKey,
		types.NewAddressByStr(BoltContractAddress), "Get", pb.String(string(randKey)))
	require.Nil(t, err)
	queryKey.Nonce = rand.Uint64()

	receipt, err = cli.SendView(queryKey)
	require.Nil(t, err)
	require.Equal(t, pb.Receipt_FAILED, receipt.Status)
	require.NotEqual(t, value, string(receipt.Ret))

	// test sending write-ledger tx to SendTransaction api
	hash, err := cli.SendTransaction(tx, nil)
	require.Nil(t, err)

	ret, err := cli.GetReceipt(hash)
	require.Nil(t, err)
	require.Equal(t, hash, ret.TxHash.String())
	require.Equal(t, pb.Receipt_SUCCESS, ret.Status, string(ret.Ret))

	// test sending read-ledger tx to SendView api
	view, err := genContractTransaction(pb.TransactionData_BVM, privKey,
		types.NewAddressByStr(BoltContractAddress), "Get", pb.String(string(randKey)))
	require.Nil(t, err)
	view.Nonce = rand.Uint64()

	receipt, err = cli.SendView(view)
	require.Nil(t, err)
	require.Equal(t, value, string(receipt.Ret))
}

func TestChainClient_GetTransaction(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	privKey1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	to, err := privKey1.PublicKey().Address()
	require.Nil(t, err)

	data := &pb.TransactionData{
		Amount: "10",
	}

	payload, err := data.Marshal()
	require.Nil(t, err)

	expectedTx := &pb.BxhTransaction{
		From:      from,
		To:        to,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
	}
	failReceipt, err := cli.SendTransactionWithReceipt(expectedTx, nil)
	require.Nil(t, err)
	require.True(t, strings.Contains(string(failReceipt.GetRet()), ErrBalance), string(failReceipt.GetRet()))

	// make sure account have sufficient balance
	err = transferFromAdmin(cli, from, defaultBalance)
	require.Nil(t, err)

	receipt, err := cli.SendTransactionWithReceipt(expectedTx, nil)
	require.Nil(t, err)

	t.Run("GetTransaction", func(t *testing.T) {
		actualTx, err := cli.GetTransaction(receipt.TxHash.String())
		require.Nil(t, err)
		require.Equal(t, expectedTx.SignHash(), actualTx.Tx.SignHash())
	})

	t.Run("GetTransactionByBlockHashAndIndex", func(t *testing.T) {
		tx, err := cli.GetTransaction(receipt.TxHash.String())
		require.Nil(t, err)
		meta := tx.GetTxMeta()
		actualTx, err := cli.GetTransactionByBlockHashAndIndex(common.BytesToHash(meta.BlockHash).String(), meta.Index)
		require.Nil(t, err)
		require.Equal(t, expectedTx.SignHash(), actualTx.Tx.SignHash())

	})

	t.Run("GetTransactionByBlockNumberAndIndex", func(t *testing.T) {
		tx, err := cli.GetTransaction(receipt.TxHash.String())
		require.Nil(t, err)
		meta := tx.GetTxMeta()
		actualTx, err := cli.GetTransactionByBlockNumberAndIndex(meta.BlockHeight, meta.Index)
		require.Nil(t, err)
		require.Equal(t, expectedTx.SignHash(), actualTx.Tx.SignHash())
	})
}

func transferFromAdmin(client *ChainClient, address *types.Address, amount string) error {
	keyPath := filepath.Join("testdata/node1", keyFile)
	adminPrivKey, err := asym.RestorePrivateKey(keyPath, keyPassword)
	if err != nil {
		return err
	}

	adminFrom, err := adminPrivKey.PublicKey().Address()
	if err != nil {
		return err
	}

	data := &pb.TransactionData{
		Amount: amount,
	}
	payload, err := data.Marshal()
	if err != nil {
		return err
	}

	tx := &pb.BxhTransaction{
		From:      adminFrom,
		To:        address,
		Timestamp: time.Now().UnixNano(),
		Payload:   payload,
	}

	adminNonce, err := client.GetPendingNonceByAccount(adminFrom.String())
	if err != nil {
		return err
	}

	ret, err := client.SendTransactionWithReceipt(tx, &TransactOpts{
		From:    adminFrom.String(),
		Nonce:   atomic.AddUint64(&adminNonce, 1) - 1,
		PrivKey: adminPrivKey,
	})
	if err != nil {
		return err
	}
	if ret.Status != pb.Receipt_SUCCESS {
		return fmt.Errorf(string(ret.Ret))
	}
	return nil
}

func TestChainClient_GetChainMeta(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	chainMeta, err := cli.GetChainMeta()
	require.Nil(t, err)
	require.True(t, chainMeta.GetHeight() > 0)
}

func TestChainClient_GetSigns(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	block, err := cli.GetBlock("", pb.GetBlockRequest_LATEST, false)
	require.Nil(t, err)
	rep, err := cli.GetMultiSigns(fmt.Sprintf("%d", block.Height()), pb.GetSignsRequest_MULTI_BLOCK_HEADER)
	require.Nil(t, err)
	require.NotEmpty(t, rep.Sign)

	//time1 := time.Now()
	//rep, err = cli.GetSigns("1356:0xF9e13c4266e96e4C0Da03eca961e609f792CB3aa:mychannel&transfer-1356:0x7d8B9C5c5D192A5425402369c5127B389f5CfE95:mychannel&transfer-1",
	//	pb.GetSignsRequest_MULTI_IBTP_REQUEST)
	//timeMulti := time.Since(time1).Milliseconds()
	//fmt.Printf("2==== time: %d\n", timeMulti)
	//require.Nil(t, err)
	//for k, v := range rep.Sign {
	//	fmt.Printf("K: %s, V: %s\n", k, string(v))
	//}
	//
	//time2 := time.Now()
	//rep, err = cli.GetTssSigns("1356:0xe15F3277214e280e47e5D018940dd78D0Fcb50A8:mychannel&transfer-1356:0x99e1e5F3D3664a094E6CeFb24ab68E1953E40c79:mychannel&transfer-1",
	//	pb.GetSignsRequest_TSS_IBTP_REQUEST,
	//	[]byte("1,2,3,4"))
	//timeTss := time.Since(time2).Milliseconds()
	//fmt.Printf("3==== time: %d\n", timeTss)
	//require.Nil(t, err)
	//for k, v := range rep.Sign {
	//	fmt.Printf("K: %s, V: %s\n", k, string(v))
	//}
}

func TestChainClient_GetAccountBalance(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)
	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)
	res, err := cli.GetAccountBalance(address.String())
	require.Nil(t, err)
	meta := &Account{}
	err = json.Unmarshal(res.Data, meta)
	require.Nil(t, err)
	require.Equal(t, big.NewInt(0), meta.Balance)

	err = transferFromAdmin(cli, address, defaultBalance)
	require.Nil(t, err)
	res, err = cli.GetAccountBalance(address.String())
	require.Nil(t, err)
	meta1 := &Account{}
	err = json.Unmarshal(res.Data, meta1)
	require.Nil(t, err)
	require.Equal(t, defaultBalance, meta1.Balance.String())
}

func TestChainClient_GetTPS(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	tx, err := genContractTransaction(pb.TransactionData_BVM, privKey,
		types.NewAddressByStr(BoltContractAddress), "Set", pb.String(string("a")), pb.String("1"))
	require.Nil(t, err)

	err = tx.Sign(privKey)
	require.Nil(t, err)

	_, err = cli.sendTransactionWithReceipt(tx, nil)
	require.Nil(t, err)

	meta0, err := cli.GetChainMeta()
	require.Nil(t, err)

	for i := 0; i < 10; i++ {
		tx, err = genContractTransaction(pb.TransactionData_BVM, privKey,
			types.NewAddressByStr(BoltContractAddress), "Set", pb.String(string("a")), pb.String("1"))
		require.Nil(t, err)

		err = tx.Sign(privKey)
		require.Nil(t, err)

		_, err = cli.sendTransaction(tx, nil)
		require.Nil(t, err)

		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)

	meta1, err := cli.GetChainMeta()
	require.Nil(t, err)

	tps, err := cli.GetTPS(meta0.Height, meta1.Height)
	require.Nil(t, err)
	require.True(t, tps > 0)
}

func TestChainClient_GetChainID(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	require.Nil(t, err)

	res, err := cli.GetChainID()
	require.Nil(t, err)
	require.Equal(t, uint64(1356), res)
}

func genContractTransaction(
	vmType pb.TransactionData_VMType, privateKey crypto.PrivateKey,
	address *types.Address, method string, args ...*pb.Arg) (*pb.BxhTransaction, error) {
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

	payload, err := td.Marshal()
	if err != nil {
		return nil, err
	}

	tx := &pb.BxhTransaction{
		From:      from,
		To:        address,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
	}

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	tx.TransactionHash = tx.Hash()

	return tx, nil
}
