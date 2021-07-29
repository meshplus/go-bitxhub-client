package rpcx

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/meshplus/bitxhub-model/constant"

	"github.com/stretchr/testify/require"
)

func TestChainClient_DeployXVMContract(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	contract, err := ioutil.ReadFile("./testdata/example.wasm")
	require.Nil(t, err)

	_, err = cli.DeployContract(contract, nil)
	require.Nil(t, err)
}

func TestChainClient_InvokeXVMContract(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	contract, err := ioutil.ReadFile("./testdata/example.wasm")
	require.Nil(t, err)

	addr, err := cli.DeployContract(contract, nil)
	require.Nil(t, err)

	result, err := cli.InvokeXVMContract(addr, "a", nil, Int32(1), Int32(2))
	require.Nil(t, err)
	require.True(t, CheckReceipt(result))
	require.Equal(t, "336", string(result.Ret))
}

func TestChainClient_InvokeBVMContract(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)

	result, err := cli.InvokeBVMContract(constant.StoreContractAddr.Address(), "Set", nil, String("a"), String("10"))
	require.Nil(t, err)
	require.Nil(t, result.Ret)
	res, err := cli.InvokeBVMContract(constant.StoreContractAddr.Address(), "Get", nil, String("a"))
	require.Nil(t, err)
	require.Equal(t, string(res.Ret), "10")
}

func TestSetProxy_InvokeBVMContract(t *testing.T) {
	cli, err := Cli2()
	require.Nil(t, err)
	result, err := cli.InvokeBVMContract(constant.EthHeaderMgrContractAddr.Address(), "SetEscrowAddr",
		nil, String("0x2EE82c6830dFd73315aB1Fecad20972aDB8dfD4d"))
	require.Nil(t, err)
	fmt.Println(result)
}
