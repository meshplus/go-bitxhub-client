Go Bitxhub Client
=====
![build](https://github.com/meshplus/go-bitxhub-client/workflows/build/badge.svg)
[![codecov](https://codecov.io/gh/meshplus/go-bitxhub-client/branch/master/graph/badge.svg)](https://codecov.io/gh/meshplus/go-bitxhub-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/meshplus/go-bitxhub-client)](https://goreportcard.com/report/github.com/meshplus/go-bitxhub-client)

This SDK enables Go developers to build solutions that interact with BitXHub.

## Getting started
Obtain the client SDK packages for BitXHub.
```shell
go get github.com/meshplus/go-bitxhub-client
```

You're good to go, happy coding! Check out the examples for usage demonstrations.

### Documentation

SDK documentation can be viewed at [GoDoc](https://github.com/meshplus/go-bitxhub-client/wiki/Go-SDK%E4%BD%BF%E7%94%A8%E6%96%87%E6%A1%A3).

### Examples

- [RPC Test](./rpcx_test.go): Basic example that uses SDK to query and execute transaction.
- [Block Test](./block_test.go): Basic example that uses SDK to query blocks.
- [Contract Test](./contract_test.go): Basic example that uses SDK to deploy and invoke contract.
- [Subscribe Test](./subscribe_test.go): An example that uses SDK to subscribe the block event.
- [Sync Test](./sync_test.go): An example that uses SDK to sync the merkle wrapper.

## Client SDK
You should start [BitXHub](https://github.com/meshplus/bitxhub) before using SDK.

### Running the test
Obtain the client SDK packages for BitXHub.
```shell script
git clone https://github.com/meshplus/go-bitxhub-client.git
```
```shell script
# In the BitXHub SDK Go directory
cd go-bitxhub-client/

# make depend
go mod tidy

# Running test
make test
```

### Contributing
See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

[Apache-2.0](https://github.com/meshplus/go-bitxhub-client/blob/master/LICENSE)