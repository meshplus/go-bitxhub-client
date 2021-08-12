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
	cli, _, from, _ := prepareKeypair(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sendInterchaintx(t, cli, *from)

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

func sendInterchaintx(t *testing.T, cli *ChainClient, from types.Address) {
	validators, err := ioutil.ReadFile("./testdata/single_validator")
	require.Nil(t, err)

	proof, err := ioutil.ReadFile("./testdata/proof_1.0.0_rc")
	require.Nil(t, err)

	rawPubKey, err := cli.privateKey.PublicKey().Bytes()
	require.Nil(t, err)
	pubKey := base64.StdEncoding.EncodeToString(rawPubKey)

	// regiter approve
	// you should put your bitxhub/scripts/build/node1/key.json to testdata/node1/key.json.
	adminKey1 := filepath.Join("./testdata/node1", "key.json")
	adminKey2 := filepath.Join("./testdata/node2", "key.json")
	adminKey3 := filepath.Join("./testdata/node3", "key.json")
	adminCli1 := getAdminCli(t, adminKey1)
	adminCli2 := getAdminCli(t, adminKey2)
	adminCli3 := getAdminCli(t, adminKey3)

	priAdmin1, err := asym.RestorePrivateKey(adminKey1, "bitxhub")
	require.Nil(t, err)
	fromAdmin1, err := priAdmin1.PublicKey().Address()
	require.Nil(t, err)

	require.Nil(t, err)
	priAdmin2, err := asym.RestorePrivateKey(adminKey2, "bitxhub")
	require.Nil(t, err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	require.Nil(t, err)
	priAdmin3, err := asym.RestorePrivateKey(adminKey3, "bitxhub")
	require.Nil(t, err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	require.Nil(t, err)

	// init registry first
	adminDid := genUniqueRelaychainDID(fromAdmin1.String())
	args := []*pb.Arg{
		pb.String(adminDid),
	}
	ret, err := adminCli1.InvokeBVMContract(constant.MethodRegistryContractAddr.Address(), "Init", nil, args...)
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))
	// set admin for method registry for other nodes
	args = []*pb.Arg{
		pb.String(adminDid),
		pb.String(genUniqueRelaychainDID(fromAdmin2.String())),
	}
	ret, err = adminCli1.InvokeBVMContract(constant.MethodRegistryContractAddr.Address(), "AddAdmin", nil, args...)
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))

	args = []*pb.Arg{
		pb.String(adminDid),
		pb.String(genUniqueRelaychainDID(fromAdmin3.String())),
	}
	ret, err = adminCli1.InvokeBVMContract(constant.MethodRegistryContractAddr.Address(), "AddAdmin", nil, args...)
	require.Nil(t, err)
	require.True(t, ret.IsSuccess(), string(ret.Ret))

	// register src appchain
	appchainMethod := fmt.Sprintf("appchain%s", from.String())
	fmt.Println("appchainMethod", appchainMethod)
	r, err := cli.InvokeBVMContract(
		constant.AppchainMgrContractAddr.Address(),
		"Register", nil,
		pb.String(appchainMethod),
		pb.String(docAddr), pb.String(docHash),
		String(string(validators)), String("rbft"), String("hyperchain"), String("hpc"),
		String("hyperchain"), String("1.0.0"), String(pubKey),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
	proposalId := gjson.Get(string(r.Ret), "proposal_id").String()

	// vote for appchain register
	vote(t, adminCli1, adminCli2, adminCli3, proposalId)

	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	dstRawPubKey, err := privKey.PublicKey().Bytes()
	require.Nil(t, err)
	dstPubKey := base64.StdEncoding.EncodeToString(dstRawPubKey)
	to, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	dstAppchainMethod := fmt.Sprintf("appchain%s", to.String())
	// register dst appchain
	fmt.Println("dstAppchainMethod", dstAppchainMethod)
	r, err = cli.InvokeBVMContract(
		constant.AppchainMgrContractAddr.Address(),
		"Register", nil,
		pb.String(dstAppchainMethod),
		pb.String(docAddr), pb.String(docHash),
		String(string(validators)), String("rbft"), String("hyperchain"), String("hpc"),
		String("hyperchain"), String("1.0.0"), String(dstPubKey),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
	proposalId = gjson.Get(string(r.Ret), "proposal_id").String()

	// vote for appchain register
	vote(t, adminCli1, adminCli2, adminCli3, proposalId)

	appchainID := fmt.Sprintf("did:bitxhub:%s:.", appchainMethod)
	dstAppchainID := fmt.Sprintf("did:bitxhub:%s:.", dstAppchainMethod)
	// deploy rule for validation
	proposalId = deployRule(t, cli, appchainID)

	// vote for rule register
	vote(t, adminCli1, adminCli2, adminCli3, proposalId)

	ibtp := getIBTP(t, appchainID, dstAppchainID, 1, pb.IBTP_INTERCHAIN, proof)

	tx, _ := cli.GenerateIBTPTx(ibtp)
	tx.Extra = proof
	r, err = cli.SendTransactionWithReceipt(tx, nil)
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
		"RegisterRule", nil, pb.String(appchainID), pb.String(contractAddr.String()))
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
