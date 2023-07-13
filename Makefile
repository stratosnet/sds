BUILDDIR ?= $(CURDIR)/target

LEDGER_ENABLED ?= false

TEST_DOCKER_REPO=sds-rsnode
BUILD_ARGS=-o $(BUILDDIR)

ifeq ($(LEDGER_ENABLED),true)
  BUILD_FLAGS = -tags "ledger"
endif

build: go.sum $(BUILDDIR)/
	go $@ $(BUILD_ARGS) $(BUILD_FLAGS) ./cmd/...

install: build go.sum $(BUILDDIR)/
	go $@ ./cmd/...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

build-linux: go.sum
	GOOS=linux GOARCH=amd64 $(MAKE) build

build-mac: go.sum
	GOOS=darwin GOARCH=amd64 $(MAKE) build

build-windows: go.sum
	GOOS=windows GOARCH=amd64 $(MAKE) build

build-docker:
	@docker build -f Dockerfile -t ${TEST_DOCKER_REPO}:$(shell git rev-parse --short HEAD) --build-arg uid=$(shell id -u) --build-arg gid=$(shell id -g) .
	@docker tag ${TEST_DOCKER_REPO}:$(shell git rev-parse --short HEAD) ${TEST_DOCKER_REPO}:$(shell git rev-parse --abbrev-ref HEAD | sed 's#/#_#g')
	@docker tag ${TEST_DOCKER_REPO}:$(shell git rev-parse --short HEAD) ${TEST_DOCKER_REPO}:latest

update:
	go mod vendor

coverage:
	go test ./... -coverprofile cover.out -coverpkg=./...
	go tool cover -html cover.out -o target/cover.html
	go tool cover -func cover.out | grep total:
	rm cover.out

lint:
	golangci-lint run

.PHONY: build-linux build-mac build-docker build clean coverage lint
