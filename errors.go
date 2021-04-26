package rpcx

import (
	"errors"
	"fmt"
)

var (
	// error type which can be fixed by retrying
	ErrRecoverable = errors.New("recoverable error")

	// error type which tx format is invalid to send
	ErrReconstruct = errors.New("invalid tx format error")

	// set ibtp and normal nonce at the same time
	ErrIllegalNonceSet = fmt.Errorf("%w: can't set ibtp nonce and normal nonce at the same time", ErrReconstruct)

	// signature for tx is invalid
	ErrSignTx = fmt.Errorf("%w: sign for transaction invalid", ErrReconstruct)

	// network problem received from grpc
	ErrBrokenNetwork = fmt.Errorf("%w: grpc broker error", ErrRecoverable)
)
