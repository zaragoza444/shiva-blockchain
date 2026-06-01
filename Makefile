.PHONY: build test docker run-node run-testnet clean

build:
	go build -o bin/onexd ./cmd/onexd
	go build -o bin/onex ./cmd/onex
	go build -o bin/onex-bridge ./cmd/onex-bridge
	go build -o bin/onex-ai ./cmd/onex-ai

test:
	go test ./internal/...

docker:
	docker compose build

run-node: build
	./bin/onexd -datadir ./data/node1 -api :8545 -listen :30303 -seeds ./configs/seeds-mainnet.json

run-testnet: build
	./bin/onexd -datadir ./data/testnet -genesis ./configs/genesis-testnet.json -api :8547 -listen :30305 -mine -faucet -miner a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890

clean:
	rm -rf bin data
