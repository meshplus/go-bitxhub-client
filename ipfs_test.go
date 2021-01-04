package rpcx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainClient_IPFS(t *testing.T) {
	cli, err := Cli() // "http://localhost:5001"
	require.Nil(t, err)
	res, err := cli.IPFSPutFromLocal("./testdata/ipfs.json")
	require.Nil(t, err)
	fmt.Println(string(res.Data))
	_, err = cli.IPFSGet("/ipfs/" + string(res.Data))
	require.Nil(t, err)
	_, err = cli.IPFSGetToLocal("/ipfs/"+string(res.Data), "./testdata/ipfs-get.json")
	require.Nil(t, err)
}
