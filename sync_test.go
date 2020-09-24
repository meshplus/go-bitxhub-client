package rpcx

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
)

func TestChainClient_GetBlockHeader(t *testing.T) {
	cli, privKey, from, to := prepareKeypair(t)

	sendNormal(t, cli, from, to, privKey)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan *pb.BlockHeader)
	require.Nil(t, cli.GetBlockHeader(ctx, 1, 2, ch))

	for {
		select {
		case header, ok := <-ch:
			require.Equal(t, true, ok)

			require.Equal(t, header.Number, uint64(1))
			if err := cli.Stop(); err != nil {
				return
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func TestChainClient_GetInterchainTxWrappers(t *testing.T) {
	cli, _, from, to := prepareKeypair(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sendInterchaintx(t, cli, from, to)

	meta, err := cli.GetChainMeta()
	require.Nil(t, err)

	ch := make(chan *pb.InterchainTxWrappers, 10)
	require.Nil(t, cli.GetInterchainTxWrappers(ctx, to.String(), meta.Height, meta.Height+100, ch))

	for {
		select {
		case wrappers, ok := <-ch:
			require.Equal(t, true, ok)

			require.NotNil(t, wrappers.InterchainTxWrappers[0])
			wrapper := wrappers.InterchainTxWrappers[0]
			require.GreaterOrEqual(t, wrapper.Height, meta.Height)
			if err := cli.Stop(); err != nil {
				return
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func prepareKeypair(t *testing.T) (cli *ChainClient, privKey crypto.PrivateKey, from, to types.Address) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)
	privKey1, err := asym.GenerateKeyPair(crypto.Secp256k1)
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

func sendNormal(t *testing.T, cli *ChainClient, from, to types.Address, privKey crypto.PrivateKey) {
	tx := &pb.Transaction{
		From: from,
		To:   to,
		Data: &pb.TransactionData{
			Amount: 10,
		},
		Timestamp: time.Now().UnixNano(),
	}

	//require.Nil(t, tx.Sign(privKey))

	hash, err := cli.SendTransaction(tx, nil)
	require.Nil(t, err)
	require.EqualValues(t, 66, len(hash))
}

func sendInterchaintx(t *testing.T, cli *ChainClient, from, to types.Address) {
	validators, err := ioutil.ReadFile("./testdata/single_validator")
	require.Nil(t, err)

	proof, err := ioutil.ReadFile("./testdata/proof_1.0.0_rc")
	require.Nil(t, err)

	pubKey, err := cli.privateKey.PublicKey().Bytes()
	require.Nil(t, err)

	// register and audit appchain
	_, err = cli.InvokeBVMContract(
		AppchainMgrContractAddr,
		"Register", nil, String(string(validators)),
		Int32(1), String("fabric"), String("fab"),
		String("fabric"), String("1.0.0"), String(string(pubKey)),
	)
	require.Nil(t, err)

	_, err = cli.InvokeBVMContract(
		AppchainMgrContractAddr,
		"Audit", nil, String(from.String()),
		Int32(1), String("Audit passed"))
	require.Nil(t, err)

	ruleAddr := "0x00000000000000000000000000000000000000a1"
	_, err = cli.InvokeContract(pb.TransactionData_BVM, RuleManagerContractAddr,
		"RegisterRule", nil, String(from.String()), String(ruleAddr))

	ibtp := getIBTP(t, from.String(), to.String(), 1, pb.IBTP_INTERCHAIN, proof)

	b, err := ibtp.Marshal()
	require.Nil(t, err)

	_, err = cli.InvokeContract(pb.TransactionData_BVM, InterchainContractAddr,
		"HandleIBTP", &TransactOpts{
			From:      ibtpAccount(ibtp),
			IBTPNonce: ibtp.Index,
		}, Bytes(b))
	pl := &pb.InvokePayload{
		Method: "HandleIBTP",
		Args:   []*pb.Arg{Bytes(b)}[:],
	}

	data, err := pl.Marshal()
	require.Nil(t, err)

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  pb.TransactionData_BVM,
		Payload: data,
	}

	tx := &pb.Transaction{
		From:      from,
		To:        InterchainContractAddr,
		Data:      td,
		Timestamp: time.Now().UnixNano(),
		Nonce:     uint64(rand.Int63()),
		Extra:     proof,
	}

	err = tx.Sign(cli.privateKey)
	require.Nil(t, err)

	_, err = cli.sendTransactionWithReceipt(tx, nil)
	require.Nil(t, err)
}

func deployRule(t *testing.T, cli *ChainClient, from types.Address) {
	contract, err := ioutil.ReadFile("./testdata/simple_rule.wasm")
	require.Nil(t, err)

	contractAddr, err := cli.DeployContract(contract, nil)
	require.Nil(t, err)

	_, err = cli.InvokeBVMContract(
		RuleManagerContractAddr,
		"RegisterRule", nil,
		String(from.String()),
		String(contractAddr.String()))
	require.Nil(t, err)
}

func getIBTP(t *testing.T, from, to string, index uint64, typ pb.IBTP_Type, proof []byte) *pb.IBTP {
	content := &pb.Content{
		SrcContractId: "mychannel&transfer",
		DstContractId: "mychannel&transfer",
		Func:          "interchainCharge",
		Args:          [][]byte{[]byte("Alice"), []byte("Alice"), []byte("1")},
		Callback:      "interchainConfirm",
	}

	bytes, _ := content.Marshal()

	payload := &pb.Payload{
		Encrypted: false,
		Content:   bytes,
	}

	ibtppd, _ := payload.Marshal()
	proofHash := sha256.Sum256(proof)

	return &pb.IBTP{
		From:      from,
		To:        to,
		Payload:   ibtppd,
		Index:     index,
		Type:      typ,
		Timestamp: time.Now().UnixNano(),
		Proof:     proofHash[:],
	}
}

func ibtpAccount(ibtp *pb.IBTP) string {
	return fmt.Sprintf("%s-%s", ibtp.From, ibtp.To)
}
