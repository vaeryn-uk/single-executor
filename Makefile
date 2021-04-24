.PHONY: build
build: built/binary built/chain

.PHONY: clean
clean:
	rm -rf built/*

built/binary: cmd/binary
	go build -o built/binary cmd/binary/main.go

built/chain: cmd/chain
	go build -o built/chain cmd/chain/main.go