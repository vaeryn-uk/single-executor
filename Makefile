GOBUILDER=docker run --rm -v="$$(pwd):/app" -v "$$(pwd)/.gocache:/.cache" --workdir=/app --user="$$(id -u)" golang
WATCHDOGFILES=$(shell find . -path \*watchdog\*.go -print)
UTILFILES=$(shell find . -path \*util\*.go -print)
WATCHDOGCONFIG=$(shell find . -path \*watchdog\*.yaml -print)
INIT=.gocache .env built/flags

.PHONY: build
build: built/binary built/chain built/watchdog built/demoserver

.PHONY: demo
demo: build built/flags/validator-image
	docker-compose build

.PHONY: run-demo
run-demo: demo
	docker-compose up

.PHONY: clean
clean:
	rm -rf .gocache
	rm -rf built

built/binary: $(UTILFILES) | $(INIT)
	$(GOBUILDER) go build -o built/binary cmd/binary/main.go

built/chain: $(UTILFILES) | $(INIT)
	$(GOBUILDER) go build -o built/chain cmd/chain/main.go

built/watchdog: $(UTILFILES) $(WATCHDOGFILES) | $(INIT)
	$(GOBUILDER) go build -o built/watchdog cmd/watchdog/main.go

built/demoserver: $(UTILFILES) | $(INIT)
	$(GOBUILDER) go build -o built/demoserver cmd/demoserver/main.go

built/flags/validator-image: $(WATCHDOGCONFIG) docker/validator/Dockerfile built/watchdog built/binary | $(INIT)
	docker build -t single-executor-validator -f docker/validator/Dockerfile .
	touch built/flags/validator-image

built/flags:
	mkdir -p built/flags

.gocache:
	mkdir -p .gocache

.env:
	cp .env.dist .env
