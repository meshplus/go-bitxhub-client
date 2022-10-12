package rpcx

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChainClient_SubscribeAudit(t *testing.T) {
	adminCli1, adminCli2, adminCli3, appchainCli, nodeCli, appchainAdminAddr, nodeAccount := prepare(t)
	appchainID := "appchain111"
	registerAppchain(t, adminCli1, adminCli2, adminCli3, appchainCli, appchainID, appchainID, appchainAdminAddr)
	registerNode(t, adminCli1, adminCli2, adminCli3, nodeAccount, "审计节点", appchainID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := nodeCli.SubscribeAudit(ctx, pb.AuditSubscriptionRequest_AUDIT_NODE, 1, nil)
	require.Nil(t, err)

	// GetBlocks
	_, err = nodeCli.GetBlocks(1, 1)
	require.NotNil(t, err)

	// GetBlock
	_, err = nodeCli.GetBlock("1", pb.GetBlockRequest_HEIGHT)
	require.NotNil(t, err)

	// GetChainStatus
	_, err = nodeCli.GetChainStatus()
	require.NotNil(t, err)

	updateAppchain(t, adminCli1, adminCli2, adminCli3, appchainCli, appchainID, appchainID, appchainAdminAddr)

	for {
		select {
		case infoData, ok := <-c:
			require.Equal(t, true, ok)
			require.NotNil(t, infoData)
			auditTxInfo := infoData.(*pb.AuditTxInfo)
			if !auditTxInfo.Tx.IsIBTP() {
				data := &pb.TransactionData{}
				err = data.Unmarshal(auditTxInfo.Tx.GetPayload())
				require.Nil(t, err)
				require.Equal(t, pb.TransactionData_INVOKE, data.Type)
				require.Equal(t, pb.TransactionData_BVM, data.VmType)

				payload := &pb.InvokePayload{}
				err = payload.Unmarshal(data.Payload)
				require.Nil(t, err)
				fmt.Printf("================ invoke info: \n"+
					"from: %s\n"+
					"to: %s\n"+
					"method: %s\n"+
					"args: %v\n",
					auditTxInfo.Tx.From.String(),
					auditTxInfo.Tx.To.String(),
					payload.Method,
					payload.Args,
				)
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func prepare(t *testing.T) (*ChainClient, *ChainClient, *ChainClient, *ChainClient, *ChainClient, string, string) {
	path1 := "./testdata/node1/key.json"
	keyPath1 := filepath.Join(path1)
	pri1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	require.Nil(t, err)
	adminCli1, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(pri1),
	)
	require.Nil(t, err)

	path2 := "./testdata/node2/key.json"
	keyPath2 := filepath.Join(path2)
	pri2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	require.Nil(t, err)
	adminCli2, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(pri2),
	)
	require.Nil(t, err)

	path3 := "./testdata/node3/key.json"
	keyPath3 := filepath.Join(path3)
	pri3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	require.Nil(t, err)
	adminCli3, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(pri3),
	)
	require.Nil(t, err)

	path4 := "./testdata/key.json"
	keyPath4 := filepath.Join(path4)
	pri4, err := asym.RestorePrivateKey(keyPath4, "bitxhub")
	require.Nil(t, err)
	appchainCli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(pri4),
	)
	require.Nil(t, err)

	path5 := "./testdata/key1.json"
	keyPath5 := filepath.Join(path5)
	nodeKey, err := asym.RestorePrivateKey(keyPath5, "bitxhub")
	require.Nil(t, err)
	nodeCli, err := New(
		WithNodesInfo(cfg.nodesInfo...),
		WithLogger(cfg.logger),
		WithPrivateKey(nodeKey),
	)
	require.Nil(t, err)

	adminAddr1, err := adminCli1.privateKey.PublicKey().Address()
	require.Nil(t, err)
	appchainAddr, err := appchainCli.privateKey.PublicKey().Address()
	require.Nil(t, err)
	nodeAddr, err := nodeCli.privateKey.PublicKey().Address()
	require.Nil(t, err)

	nonce, err := adminCli1.GetPendingNonceByAccount(adminAddr1.String())
	require.Nil(t, err)
	transfer(t, adminCli1, appchainAddr, 1000000000000000, &TransactOpts{
		Nonce: nonce,
		From:  appchainAddr.String(),
	})
	transfer(t, adminCli1, nodeAddr, 1000000000000000, &TransactOpts{
		Nonce: nonce + 1,
		From:  nodeAddr.String(),
	})

	return adminCli1, adminCli2, adminCli3, appchainCli, nodeCli, appchainAddr.Address, nodeAddr.Address
}

func registerAppchain(t *testing.T, adminCli1, adminCli2, adminCli3, appchainCLi *ChainClient, chainID, chainName, appchainAdmin string) {
	r, err := appchainCLi.InvokeBVMContract(constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", nil,
		String(chainID),
		String(chainName),
		Bytes(nil),
		String("ETH"),
		Bytes(nil),
		String("broker"),
		String("des"),
		String(HappyRuleAddr),
		String("url"),
		String(appchainAdmin),
		String("reason"),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
	proposalId := gjson.Get(string(r.Ret), "proposal_id").String()

	vote(t, adminCli1, adminCli2, adminCli3, proposalId)
}

func registerNode(t *testing.T, adminCli1, adminCli2, adminCli3 *ChainClient, nodeAccount, nodeName, appchainID string) {
	r, err := adminCli1.InvokeBVMContract(constant.NodeManagerContractAddr.Address(), "RegisterNode", nil,
		String(nodeAccount),
		String("nvpNode"),
		String(""),
		Uint64(0),
		String(nodeName),
		String(appchainID),
		String("reason"),
	)
	require.Nil(t, err)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
	proposalId := gjson.Get(string(r.Ret), "proposal_id").String()

	vote(t, adminCli1, adminCli2, adminCli3, proposalId)
}

func updateAppchain(t *testing.T, adminCli1, adminCli2, adminCli3, appchainCLi *ChainClient, chainID, chainName, appchainAdmin string) {
	r, err := appchainCLi.InvokeBVMContract(constant.AppchainMgrContractAddr.Address(), "UpdateAppchain", nil,
		String(chainID),
		String(chainName+"111"),
		String("desc"),
		Bytes(nil),
		String(appchainAdmin),
		String("reason"),
	)
	require.Nil(t, err)
	require.Equal(t, true, r.IsSuccess(), string(r.Ret))
	proposalId := gjson.Get(string(r.Ret), "proposal_id").String()

	vote(t, adminCli1, adminCli2, adminCli3, proposalId)
}
