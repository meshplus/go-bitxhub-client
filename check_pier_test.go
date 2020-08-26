package rpcx

import (
	"fmt"
	"testing"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/stretchr/testify/require"
)

func TestChainClient_CheckMasterPier(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	ret, err := cli.CheckMasterPier("1")
	require.Nil(t, err)
	fmt.Println(ret)
	fmt.Println(ret.Data)
	resp := &pb.CheckPierResponse{}
	err = resp.Unmarshal(ret.Data)
	require.Nil(t, err)
	fmt.Println(resp)
}

func TestChainClient_SetMasterPier(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	ret, err := cli.SetMasterPier("1", "1")
	require.Nil(t, err)
	fmt.Println(ret.Data)
}

func TestChainClient_HeartBeat(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	ret, err := cli.HeartBeat("1", "1")
	require.Nil(t, err)
	fmt.Println(ret.Data)
}

func TestChainClient_CheckMasterPierRemote(t *testing.T) {
	cli, err := Cli1()
	require.Nil(t, err)
	ret, err := cli.CheckMasterPier("1")
	require.Nil(t, err)
	fmt.Println(ret)
	fmt.Println(ret.Data)
	resp := &pb.CheckPierResponse{}
	err = resp.Unmarshal(ret.Data)
	require.Nil(t, err)
	fmt.Println(resp)
}

func TestChainClient_SetMasterPierRemote(t *testing.T) {
	cli, err := Cli1()
	require.Nil(t, err)
	ret, err := cli.SetMasterPier("1", "2")
	require.Nil(t, err)
	fmt.Println(ret.Data)
}
