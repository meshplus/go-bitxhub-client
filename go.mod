module github.com/meshplus/go-bitxhub-client

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7 // indirect
	github.com/golang/mock v1.4.3
	github.com/ipfs/go-ipfs-api v0.2.0
	github.com/kr/text v0.2.0 // indirect
	github.com/meshplus/bitxhub-kit v1.2.1-0.20220325052414-bc17176c509d
	github.com/meshplus/bitxhub-model v1.2.1-0.20220412064024-c35cae241eb2
	github.com/pkg/errors v0.9.1 // indirect
	github.com/processout/grpc-go-pool v1.2.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.6.8
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20210220033124-5f55cee0dc0d // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/grpc v1.33.1
	gopkg.in/yaml.v3 v3.0.0-20200601152816-913338de1bd2 // indirect
)

replace github.com/agl/ed25519 => github.com/binance-chain/edwards25519 v0.0.0-20200305024217-f36fc4b53d43

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
