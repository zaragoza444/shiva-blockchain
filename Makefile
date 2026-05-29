.PHONY: build test docker run-node run-testnet clean

build:
	go build -o bin/shivad ./cmd/shivad
	go build -o bin/shiva ./cmd/shiva
	go build -o bin/shiva-bridge ./cmd/shiva-bridge
	go build -o bin/shiva-ai ./cmd/shiva-ai

test:
	go test ./internal/...

docker:
	docker compose build

run-node: build
	./bin/shivad -datadir ./data/node1 -api :8545 -listen :30303 -seeds ./configs/seeds-mainnet.json

run-testnet: build
	./bin/shivad -datadir ./data/testnet -genesis ./configs/genesis-testnet.json -api :8547 -listen :30305 -mine -faucet -miner a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890

clean:
	rm -rf bin data
