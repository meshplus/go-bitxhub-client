package rpcx

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainClient_DeployXVMContract(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	contract, err := ioutil.ReadFile("./testdata/example.wasm")
	require.Nil(t, err)

	_, err = cli.DeployContract(contract)
	require.Nil(t, err)
}

func TestChainClient_InvokeXVMContract(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	contract, err := ioutil.ReadFile("./testdata/example.wasm")
	require.Nil(t, err)

	addr, err := cli.DeployContract(contract)
	require.Nil(t, err)

	result, err := cli.InvokeXVMContract(addr, "a", Int32(1), Int32(2))
	require.Nil(t, err)
	require.True(t, CheckReceipt(result))
	require.Equal(t, "336", string(result.Ret))
}

func TestChainClient_InvokeBVMContract(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	result, err := cli.InvokeBVMContract(StoreContractAddr, "Set", String("a"), String("10"))
	require.Nil(t, err)
	require.Nil(t, result.Ret)
	res, err := cli.InvokeBVMContract(StoreContractAddr, "Get", String("a"))
	require.Nil(t, err)
	require.Equal(t, string(res.Ret), "10")
}
