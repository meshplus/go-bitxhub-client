package rpcx

type BrokenNetworkError struct{ message string }

func newBrokenNetworkError(err error) *BrokenNetworkError {
	e := new(BrokenNetworkError)
	e.message = err.Error()
	return e
}

func (e *BrokenNetworkError) Error() string {
	return e.message
}
