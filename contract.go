package rpcx

import (
	"fmt"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

// DeployContract let client deploy the wasm contract into BitXHub.
func (cli *ChainClient) DeployContract(contract []byte, opts *TransactOpts) (contractAddr types.Address, err error) {
	from, err := cli.privateKey.PublicKey().Address()
	if err != nil {
		return types.Address{}, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  pb.TransactionData_XVM,
		Payload: contract,
	}

	payload, err := td.Marshal()
	if err != nil {
		return types.Address{}, err
	}

	tx := &pb.Transaction{
		From:      *from,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
	}

	receipt, err := cli.sendTransactionWithReceipt(tx, opts)
	if err != nil {
		return types.Address{}, err
	}

	ret := types.Bytes2Address(receipt.GetRet())

	return *ret, nil
}

// InvokeContract let client invoke the wasm contract with specific method.
func (cli *ChainClient) InvokeContract(vmType pb.TransactionData_VMType, address types.Address, method string,
	opts *TransactOpts, args ...*pb.Arg) (*pb.Receipt, error) {
	from, err := cli.privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	pl := &pb.InvokePayload{
		Method: method,
		Args:   args[:],
	}

	data, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  vmType,
		Payload: data,
	}

	payload, err := td.Marshal()
	if err != nil {
		return nil, err
	}

	tx := &pb.Transaction{
		From:      *from,
		To:        address,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
	}

	return cli.sendTransactionWithReceipt(tx, opts)
}

func (cli *ChainClient) InvokeBVMContract(address types.Address, method string, opts *TransactOpts, args ...*pb.Arg) (*pb.Receipt, error) {
	return cli.InvokeContract(pb.TransactionData_BVM, address, method, opts, args...)
}
func (cli *ChainClient) InvokeXVMContract(address types.Address, method string, opts *TransactOpts, args ...*pb.Arg) (*pb.Receipt, error) {
	return cli.InvokeContract(pb.TransactionData_XVM, address, method, opts, args...)
}

func (cli *ChainClient) GenerateContractTx(vmType pb.TransactionData_VMType, address types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	from, err := cli.privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	pl := &pb.InvokePayload{
		Method: method,
		Args:   args[:],
	}

	data, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  vmType,
		Payload: data,
	}

	payload, err := td.Marshal()
	if err != nil {
		return nil, err
	}

	tx := &pb.Transaction{
		From:      *from,
		To:        address,
		Payload:   payload,
		Timestamp: time.Now().UnixNano(),
	}

	if err := tx.Sign(cli.privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	return tx, nil
}
