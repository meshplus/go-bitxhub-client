package rpcx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainClient_IPFSPutFromLocal(t *testing.T) {
	cli, err := Cli() // "http://localhost:5001"
	require.Nil(t, err)
	_, err = cli.IPFSPutFromLocal("./testdata/ipfs.json")
	require.Nil(t, err)
	// fmt.Println("res:", string(res.Data))
}

func TestChainClient_IPFSGet(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	_, err = cli.IPFSGet("/ipfs/QmcmRVFjFZtr3qTP48tW1eXKJgYsubuBuu6iVWpA4hrss6")
	require.Nil(t, err)
	// fmt.Println("res:", string(res.Data))
}

func TestChainClient_IPFSGetToLocal(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	_, err = cli.IPFSGetToLocal("/ipfs/QmcmRVFjFZtr3qTP48tW1eXKJgYsubuBuu6iVWpA4hrss6", "./testdata/ipfs-get.json")
	require.Nil(t, err)
}
