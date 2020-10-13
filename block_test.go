package rpcx

import (
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
)

func TestChainClient_GetBlock(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	block, err := cli.GetBlock("1", pb.GetBlockRequest_HEIGHT)
	require.Nil(t, err)
	require.Equal(t, block.BlockHeader.Number, uint64(1))
}

func TestChainClient_GetChainStatus(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	res, err := cli.GetChainStatus()
	require.Nil(t, err)
	require.NotNil(t, res)
}

func TestChainClient_GetBlocks(t *testing.T) {
	cli, err := Cli()
	require.Nil(t, err)
	blocks, err := cli.GetBlocks(1, 1)
	require.Nil(t, err)
	require.Equal(t, len(blocks.Blocks), 1)
}

func Cli() (*ChainClient, error) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	if err != nil {
		return nil, err
	}

	cli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(privKey),
	)
	return cli, err
}

func Cli1() (*ChainClient, error) {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	if err != nil {
		return nil, err
	}

	cli, err := New(
		WithNodesInfo(cfg1.nodesInfo...),
		WithLogger(cfg1.logger),
		WithPrivateKey(privKey),
	)
	return cli, err
}
