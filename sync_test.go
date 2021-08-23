package rpcx

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const (
	keyPassword              = "bitxhub"
	appchainAdminDIDPrefix   = "did:bitxhub:appchain"
	relaychainAdminDIDPrefix = "did:bitxhub:relayroot"
	docAddr                  = "/ipfs/QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
	docHash                  = "QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
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
	cli, _, addr, _ := prepareKeypair(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privKey0, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	from, err := privKey0.PublicKey().Address()
	require.Nil(t, err)

	cli0, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey0),
	)
	require.Nil(t, err)

	privKey1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	to, err := privKey1.PublicKey().Address()
	require.Nil(t, err)

	cli1, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey1),
	)
	require.Nil(t, err)

	nonce, err := cli.GetPendingNonceByAccount(addr.String())
	require.Nil(t, err)

	transfer(t, cli, from, 1000000000000000, &TransactOpts{Nonce: nonce})
	transfer(t, cli, to, 1000000000000000, &TransactOpts{Nonce: nonce + 1})

	sendInterchaintx(t, cli0, cli1)

	meta, err := cli.GetChainMeta()
	require.Nil(t, err)

	did := genUniqueAppchainDID(from.String())
	ch := make(chan *pb.InterchainTxWrappers, 10)
	require.Nil(t, cli.GetInterchainTxWrappers(ctx, did, meta.Height, meta.Height+100, ch))

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

func prepareKeypair(t *testing.T) (cli *ChainClient, privKey crypto.PrivateKey, from, to *types.Address) {
	privKey, err := asym.RestorePrivateKey(filepath.Join("testdata", "key.json"), "bitxhub")
	require.Nil(t, err)
	privKey1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	cli, err = New(
		WithNodesInfo(cfg.nodesInfo...),
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

func sendNormal(t *testing.T, cli *ChainClient, from, to *types.Address, privKey crypto.PrivateKey) {
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

	hash, err := cli.SendTransaction(tx, nil)
	require.Nil(t, err)
	require.EqualValues(t, 66, len(hash))
}

func sendInterchaintx(t *testing.T, cli0 *ChainClient, cli1 *ChainClient) {
	validators, err := ioutil.ReadFile("./testdata/single_validator")
	require.Nil(t, err)

	proof, err := ioutil.ReadFile("./testdata/proof_1.0.0_rc")
	require.Nil(t, err)

	// regiter approve
	// you should put your bitxhub/scripts/build/node1/key.json to testdata/node1/key.json.
	adminKey1 := filepath.Join("./testdata/node1", "key.json")
	adminKey2 := filepath.Join("./testdata/node2", "key.json")
	adminKey3 := filepath.Join("./testdata/node3", "key.json")
	adminCli1 := getAdminCli(t, adminKey1)
	adminCli2 := getAdminCli(t, adminKey2)
	adminCli3 := getAdminCli(t, adminKey3)

	srcRawPubKey, err := cli0.privateKey.PublicKey().Bytes()
	require.Nil(t, err)
	srcPubKey := base64.StdEncoding.EncodeToString(srcRawPubKey)
	from, err := cli0.privateKey.PublicKey().Address()

	dstRawPubKey, err := cli1.privateKey.PublicKey().Bytes()
	require.Nil(t, err)

	dstPubKey := base64.StdEncoding.EncodeToString(dstRawPubKey)
	to, err := cli1.privateKey.PublicKey().Address()
	require.Nil(t, err)

	appchain0 := "appchain" + from.String()
	appchain1 := "appchain" + to.String()

	// register src appchain
	r, err := cli0.InvokeBVMContract(
		constant.AppchainMgrContractAddr.Address(),
		"Register", nil,
		pb.String(appchain0),
		pb.String(docAddr), pb.String(docHash),
		String(string(validators)), String("rbft"), String("hyperchain"), String("hpc"),
		String("hyperchain"), String("1.0.0"), String(srcPubKey), String(""),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
	proposalId := gjson.Get(string(r.Ret), "proposal_id").String()

	//vote for appchain register
	vote(t, adminCli1, adminCli2, adminCli3, proposalId)

	// register dst appchain
	r, err = cli1.InvokeBVMContract(
		constant.AppchainMgrContractAddr.Address(),
		"Register", nil,
		pb.String(appchain1),
		pb.String(docAddr), pb.String(docHash),
		String(string(validators)), String("rbft"), String("hyperchain"), String("hpc"),
		String("hyperchain"), String("1.0.0"), String(dstPubKey), String("register"),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
	proposalId = gjson.Get(string(r.Ret), "proposal_id").String()

	// vote for appchain register
	vote(t, adminCli1, adminCli2, adminCli3, proposalId)
	// deploy rule for validation
	proposalId = deployRule(t, cli0, appchain0)

	// vote for rule register
	vote(t, adminCli1, adminCli2, adminCli3, proposalId)

	srcServiceID := fmt.Sprintf("1356:%s:%s", appchain0, from.String())
	dstServiceID := fmt.Sprintf("1356:%s:%s", appchain1, to.String())

	// register src service
	r, err = cli0.InvokeBVMContract(
		constant.ServiceMgrContractAddr.Address(),
		"Register", nil,
		pb.String(appchain0),
		pb.String(from.String()),
		pb.String("service 0"), pb.String("service"),
		String("contract invoker"), Bool(true), String(dstServiceID), Bytes(nil),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))

	// register dst service
	r, err = cli1.InvokeBVMContract(
		constant.ServiceMgrContractAddr.Address(),
		"Register", nil,
		pb.String(appchain1),
		pb.String(to.String()),
		pb.String("service 1"), pb.String("service"),
		String("contract invoker"), Bool(true), String(srcServiceID), Bytes(nil),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))

	ibtp := getIBTP(t, srcServiceID, dstServiceID, 1, pb.IBTP_INTERCHAIN, proof)

	tx, _ := cli0.GenerateIBTPTx(ibtp)
	tx.Extra = proof
	r, err = cli0.SendTransactionWithReceipt(tx, nil)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
}

func vote(t *testing.T, adminCli1 *ChainClient, adminCli2 *ChainClient, adminCli3 *ChainClient, proposalId string) {
	r, err := adminCli1.InvokeBVMContract(
		constant.GovernanceContractAddr.Address(),
		"Vote", nil, String(proposalId), String("approve"), String("reason"),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))

	r, err = adminCli2.InvokeBVMContract(
		constant.GovernanceContractAddr.Address(),
		"Vote", nil, String(proposalId), String("approve"), String("reason"),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))

	r, err = adminCli3.InvokeBVMContract(
		constant.GovernanceContractAddr.Address(),
		"Vote", nil, String(proposalId), String("approve"), String("reason"),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
}

func genUniqueAppchainDID(addr string) string {
	return fmt.Sprintf("%s%s:%s", appchainAdminDIDPrefix, addr, addr)
}

func genUniqueRelaychainDID(addr string) string {
	return fmt.Sprintf("%s:%s", relaychainAdminDIDPrefix, addr)
}

func deployRule(t *testing.T, cli *ChainClient, appchainID string) string {
	contract, err := ioutil.ReadFile("./testdata/simple_rule.wasm")
	require.Nil(t, err)

	contractAddr, err := cli.DeployContract(contract, nil)
	require.Nil(t, err)

	// register rule
	ret, err := cli.InvokeBVMContract(constant.RuleManagerContractAddr.Address(),
		"RegisterRule", nil, pb.String(appchainID), pb.String(contractAddr.String()), pb.String(""))
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))

	return gjson.Get(string(ret.Ret), "proposal_id").String()
}

func getIBTP(t *testing.T, from, to string, index uint64, typ pb.IBTP_Type, proof []byte) *pb.IBTP {
	content := &pb.Content{
		Func:     "interchainCharge",
		Args:     [][]byte{[]byte("Alice"), []byte("Alice"), []byte("1")},
		Callback: "interchainConfirm",
	}

	bytes, _ := content.Marshal()

	payload := &pb.Payload{
		Encrypted: false,
		Content:   bytes,
	}

	ibtppd, _ := payload.Marshal()
	proofHash := sha256.Sum256(proof)

	return &pb.IBTP{
		From:          from,
		To:            to,
		Payload:       ibtppd,
		Index:         index,
		Type:          typ,
		TimeoutHeight: 10,
		Proof:         proofHash[:],
	}
}

// getAdminCli returns client with admin account.
func getAdminCli(t *testing.T, keyPath string) *ChainClient {
	// you should put your bitxhub/scripts/build/node1/key.json to testdata/key.json.
	k, err := asym.RestorePrivateKey(keyPath, keyPassword)
	require.Nil(t, err)
	var cfg = &config{
		nodesInfo: []*NodeInfo{
			{Addr: "localhost:60011", EnableTLS: true, CertPath: "testdata/node1/certs/agency.cert", CommonName: "BitXHub",
				AccessCert: "testdata/node1/certs/gateway.cert", AccessKey: "testdata/node1/certs/gateway.priv"},
		},
		logger:     logrus.New(),
		privateKey: k,
	}
	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(cfg.privateKey),
	)
	return cli
}

func transfer(t *testing.T, cli *ChainClient, to *types.Address, amount uint64, opt *TransactOpts) {
	from, err := cli.privateKey.PublicKey().Address()
	require.Nil(t, err)

	data := &pb.TransactionData{
		Amount: fmt.Sprintf("%d", amount),
	}

	payload, err := data.Marshal()
	require.Nil(t, err)

	tx := &pb.BxhTransaction{
		From:      from,
		To:        to,
		Payload:   payload,
		Amount:    data.Amount,
		Timestamp: time.Now().UnixNano(),
	}

	_, err = cli.SendTransaction(tx, opt)
	require.Nil(t, err)
}
