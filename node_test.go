package rpcx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainClient_GetNetworkMeta(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	res, err := cli.GetNetworkMeta()
	require.Nil(t, err)
	require.NotNil(t, res)
}
