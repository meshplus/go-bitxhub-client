package rpcx

import (
	"testing"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
)

func TestChainClient_CheckMasterPier(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	ret, err := cli.CheckMasterPier("1")
	require.Nil(t, err)
	resp := &pb.CheckPierResponse{}
	err = resp.Unmarshal(ret.Data)
	require.Nil(t, err)
}
