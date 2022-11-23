module github.com/meshplus/go-bitxhub-client

go 1.18

require (
	github.com/Rican7/retry v0.1.0
	github.com/ethereum/go-ethereum v1.10.8
	github.com/golang/mock v1.6.0
	github.com/meshplus/bitxhub-kit v1.2.1-0.20221123035412-4519b8b90d90
	github.com/meshplus/bitxhub-model v1.2.1-0.20221031060115-cd3292575517
	github.com/meshplus/eth-kit v0.0.0-20221028095005-bdda18e64555
	github.com/processout/grpc-go-pool v1.2.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.8.0
	github.com/tidwall/gjson v1.6.8
	google.golang.org/grpc v1.50.1
)

require (
	github.com/btcsuite/btcd v0.21.0-beta // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.1 // indirect
	github.com/cbergoon/merkletree v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/lestrrat-go/file-rotatelogs v2.2.0+incompatible // indirect
	github.com/lestrrat-go/strftime v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5 // indirect
	github.com/tidwall/match v1.0.3 // indirect
	github.com/tidwall/pretty v1.0.2 // indirect
	golang.org/x/crypto v0.2.0 // indirect
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/genproto v0.0.0-20221014213838-99cd37c6964a // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/agl/ed25519 => github.com/binance-chain/edwards25519 v0.0.0-20200305024217-f36fc4b53d43

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4

replace google.golang.org/grpc => google.golang.org/grpc v1.33.0
