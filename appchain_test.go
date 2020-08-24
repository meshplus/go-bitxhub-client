package rpcx

import (
	"encoding/hex"
	"encoding/json"
	"github.com/meshplus/bitxhub-kit/crypto"
	"path/filepath"
	"testing"

	"github.com/docker/docker/pkg/testutil/assert"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	keyPassword = "bitxhub"
)

var AppChainID string

func TestAppChain_Register(t *testing.T) {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)
	var cfg = &config{
		addrs: []string{
			"localhost:60011",
		},
		logger:     logrus.New(),
		privateKey: privKey,
	}
	cli, err := New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(cfg.privateKey),
	)
	require.Nil(t, err)
	pubKey, err := privKey.PublicKey().Address()
	var pubKeyStr string = hex.EncodeToString(pubKey.Bytes())
	args := []*pb.Arg{
		String(""),                 //validators
		Int32(0),                   //consensus_type
		String("hyperchain"),       //chain_type
		String("AppChain1"),        //name
		String("Appchain for tax"), //desc
		String("1.8"),              //version
		String(pubKeyStr),          //public key
	}
	res, err := cli.InvokeBVMContract(AppchainMgrContractAddr, "Register", args...)
	require.Nil(t, err)
	appChain := &Appchain{}
	err = json.Unmarshal(res.Ret, appChain)
	require.Nil(t, err)
	require.NotNil(t, appChain.ID)
	AppChainID = appChain.ID
}

func testAppChain_Aduit(t *testing.T) {
	require.NotEqual(t, "", AppChainID)
	cli := getAdminCli(t)
	args := []*pb.Arg{
		String(AppChainID),
		Int32(1),               //audit approve
		String("Audit passed"), //desc
	}
	res, err := cli.InvokeBVMContract(AppchainMgrContractAddr, "Audit", args...)
	require.Nil(t, err)
	assert.Contains(t, string(res.Ret), "successfully")
}

// getAdminCli returns client with admin account.
func getAdminCli(t *testing.T) *ChainClient {
	// you should put your bitxhub/scripts/build/node1/key.json to testdata/key.json.

	k, err := asym.RestorePrivateKey(filepath.Join("testdata", "key.json"), keyPassword)
	require.Nil(t, err)
	var cfg = &config{
		addrs: []string{
			"localhost:60011",
		},
		logger:     logrus.New(),
		privateKey: k,
	}
	cli, err := New(
		WithAddrs(cfg.addrs),
		WithLogger(cfg.logger),
		WithPrivateKey(cfg.privateKey),
	)
	return cli
}
