package rpcx

import (
	"encoding/json"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAppChain_Register(t *testing.T) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	require.Nil(t, err)
	var cfg = &config{
		addrs: []string{
			"localhost:60011",
			"localhost:60012",
			"localhost:60013",
			"localhost:60014",
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
	args := []*pb.Arg{
		String(""),                 //validators
		Int32(0),                   //consensus_type
		String("hyperchain"),       //chain_type
		String("AppChain1"),        //name
		String("Appchain for tax"), //desc
		String("1.8"),              //version
	}
	res, err := cli.InvokeBVMContract(InterchainContractAddr, "Register", args...)
	require.Nil(t, err)
	appChain := &Appchain{}
	err = json.Unmarshal(res.Ret, appChain)
	require.Nil(t, err)
	require.NotNil(t, appChain.ID)
}
