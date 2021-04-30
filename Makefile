gobuilder=docker run --rm -v="$$(pwd):/app" -v "$$(pwd)/.gocache:/.cache" --workdir=/app --user="$$(id -u)" golang

.PHONY: build
build: .env .gocache built/binary built/chain built/watchdog built/demoserver

.PHONY: demo
demo: build
	docker-compose build

.PHONY: run-demo
run-demo: demo
	docker-compose up

.PHONY: clean
clean:
	rm -rf .gocache
	rm -rf built

.gocache:
	mkdir .gocache

.env:
	cp .env.dist .env

built/binary: cmd/binary
	$(gobuilder) go build -o built/binary cmd/binary/main.go

built/chain: cmd/chain
	$(gobuilder) go build -o built/chain cmd/chain/main.go

built/watchdog: cmd/watchdog
	$(gobuilder) go build -o built/watchdog cmd/watchdog/main.go

built/demoserver: cmd/demoserver
	$(gobuilder) go build -o built/demoserver cmd/demoserver/main.go