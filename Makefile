.PHONY: build
build: built/binary built/chain built/watchdog

.PHONY: clean
clean:
	rm -rf built/*

built/binary: cmd/binary
	go build -o built/binary cmd/binary/main.go

built/chain: cmd/chain
	go build -o built/chain cmd/chain/main.go

built/chain: cmd/watchdog
	go build -o built/watchdog cmd/watchdog/main.go