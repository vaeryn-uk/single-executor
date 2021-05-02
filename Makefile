GOBUILDER=docker-compose -f docker-compose.build.yaml run --rm gobuilder
GOBUILDER_BUILD=$(GOBUILDER) go build -mod=vendor
WATCHDOGFILES=$(shell find . -path \*watchdog\*.go -print)
UTILFILES=$(shell find . -path \*util\*.go -print)
DASHBOARDWEBSRCFILES=$(shell find . -path ./web/dashboard/src/\* -print)
WATCHDOGCONFIG=$(shell find . -path \*watchdog\*.yaml -print)
INIT=.cache/gocache .env built/flags

.PHONY: build
build: built/binary built/chain built/watchdog built/dashboard

.PHONY: demo
demo: build built/flags/validator-image web/dashboard/dist
	docker-compose build

.PHONY: run-demo
run-demo: demo
	docker-compose up

.PHONY: clean
clean:
	# Remove any local images created by our docker compose stacks.
	docker-compose down --remove-orphans --rmi local
	# Delete our tagged image if we have one.
	docker image rm single-executor-validator || true
	# Remove built/recreatable files.
	rm -rf .cache built web/dashboard/node_modules web/dashboard/dist vendor

built/binary: vendor $(UTILFILES) cmd/binary/main.go | $(INIT)
	$(GOBUILDER_BUILD) -o built/binary cmd/binary/main.go

built/chain: vendor $(UTILFILES) cmd/chain/main.go | $(INIT)
	$(GOBUILDER_BUILD) -o built/chain cmd/chain/main.go

built/watchdog: vendor $(UTILFILES) $(WATCHDOGFILES) | $(INIT)
	$(GOBUILDER_BUILD) -o built/watchdog cmd/watchdog/main.go

built/dashboard: vendor $(UTILFILES) cmd/dashboard/main.go | $(INIT)
	$(GOBUILDER_BUILD) -o built/dashboard cmd/dashboard/main.go

built/flags/validator-image: $(WATCHDOGCONFIG) docker/validator/Dockerfile built/watchdog built/binary | $(INIT)
	docker build -t single-executor-validator -f docker/validator/Dockerfile .
	touch built/flags/validator-image

web/dashboard/dist: web/dashboard/node_modules $(DASHBOARDWEBSRCFILES) web/dashboard/tsconfig.json web/dashboard/webpack.config.js
	docker-compose -f docker-compose.build.yaml run --rm dashboardwebbuilder npm run build
	touch web/dashboard/dist	# Update mtime of the directory.

web/dashboard/node_modules: web/dashboard/package.json
	docker-compose -f docker-compose.build.yaml run --rm dashboardwebbuilder npm install

# Drops you in to a terminal to manage/develop the dashboard VueJS app.
.PHONY: dashboardbuilder
dashboardbuilder:
	docker-compose -f docker-compose.build.yaml run --rm dashboardwebbuilder bash

built/flags:
	mkdir -p built/flags

.cache/gocache:
	mkdir -p .cache/gocache

vendor: | $(INIT)
	$(GOBUILDER) go mod vendor

.env:
	cp .env.dist .env
	sed -i s/_UID_/$$(id -u)/ .env
