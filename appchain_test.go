package rpcx

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAppChain_Register(t *testing.T) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
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
	fmt.Println("pubKey:", hex.EncodeToString(pubKey.Bytes())) // string(pubKey.Bytes()[:])
	// var pubKeyStr string = hex.EncodeToString(pubKey.Bytes())
	// args := []*pb.Arg{
	// 	String(""),                 //validators
	// 	Int32(0),                   //consensus_type
	// 	String("hyperchain"),       //chain_type
	// 	String("AppChain1"),        //name
	// 	String("Appchain for tax"), //desc
	// 	String("1.8"),              //version
	// 	// String(""),                 //public key
	// }
	fmt.Println("pubKey:", pubKey)
	// res, err := cli.InvokeBVMContract(AppchainMgrContractAddr, "Register", args...)
	res, err := cli.InvokeBVMContract(
		AppchainMgrContractAddr,
		"Register", String(""),
		Int32(1), String("fabric"), String("fab"),
		String("fabric"), String("1.0.0"), String(""),
	)
	require.Nil(t, err)
	appChain := &Appchain{}
	err = json.Unmarshal(res.Ret, appChain)
	fmt.Print("res.Ret:", string(res.Ret))
	require.Nil(t, err)
	require.NotNil(t, appChain.ID)
}
