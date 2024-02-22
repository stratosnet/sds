BUILDDIR ?= $(CURDIR)/target
TEST_DOCKER_REPO=sds-rsnode

QUIET ?= @
ifndef SDS_GIT_REVISION
SDS_GIT_REVISION := $(shell git log --pretty=format:"%h" --abbrev-commit -1)
endif

# variables for go.mod files
FRAMEWORK := framework
SDS_MSG := sds-msg
TX_CLIENT := tx-client
REPO_PATH := github.com/stratosnet/sds
REPLACE := replace

SDS_MSG_PATH := $(REPO_PATH)/$(SDS_MSG)
FRAMEWORK_PATH := $(REPO_PATH)/$(FRAMEWORK)
TX_CLIENT_PATH := $(REPO_PATH)/$(TX_CLIENT)

SDS_MSG_UPD := $(SDS_MSG_PATH)@$(SDS_GIT_REVISION)
FRAMEWORK_UPD := $(FRAMEWORK_PATH)@$(SDS_GIT_REVISION)
TX_CLIENT_UPD := $(TX_CLIENT_PATH)@$(SDS_GIT_REVISION)

FRAMEWORK_REPLACE := '$(REPLACE) $(FRAMEWORK_PATH) => ./$(FRAMEWORK)'
SDS_MSG_REPLACE := '$(REPLACE) $(SDS_MSG_PATH) => ./$(SDS_MSG)'
TX_CLIENT_REPLACE := '$(REPLACE) $(TX_CLIENT_PATH) => ./$(TX_CLIENT)'

FRAMEWORK_REPLACE_FROM_TX_CLIENT := '$(REPLACE) $(FRAMEWORK_PATH) => ../$(FRAMEWORK)'
SDS_MSG_REPLACE_FROM_TX_CLIENT := '$(REPLACE) $(SDS_MSG_PATH) => ../$(SDS_MSG)'

# targets
BUILD_TARGETS := build install

build: BUILD_ARGS=-o $(BUILDDIR)/

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	go $@ $(BUILD_ARGS) ./cmd/...

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

go-mod-update-sds:
ifeq ($(SDS_GIT_REVISION), LOCAL)
	$(QUIET)grep -qxF $(FRAMEWORK_REPLACE) go.mod || echo $(FRAMEWORK_REPLACE) >> go.mod
	$(QUIET)grep -qxF $(SDS_MSG_REPLACE) go.mod || echo $(SDS_MSG_REPLACE) >> go.mod
	$(QUIET)grep -qxF $(TX_CLIENT_REPLACE) go.mod || echo $(TX_CLIENT_REPLACE) >> go.mod
else
	$(QUIET)sed -i "/replace github.com\/stratosnet\/sds/d" go.mod
	$(QUIET)go get $(SDS_MSG_UPD)
	$(QUIET)go get $(FRAMEWORK_UPD)
	$(QUIET)go get $(TX_CLIENT_UPD)
	$(QUIET)go mod tidy
endif

go-mod-update-tx-client:
ifeq ($(SDS_GIT_REVISION), LOCAL)
	$(QUIET)cd tx-client &&	grep -qxF $(FRAMEWORK_REPLACE_FROM_TX_CLIENT) go.mod || echo $(FRAMEWORK_REPLACE_FROM_TX_CLIENT) >> go.mod
	$(QUIET)cd tx-client && grep -qxF $(SDS_MSG_REPLACE_FROM_TX_CLIENT) go.mod || echo $(SDS_MSG_REPLACE_FROM_TX_CLIENT) >> go.mod
else
	$(QUIET)cd tx-client && sed -i "/replace github.com\/stratosnet\/sds/d" go.mod
	$(QUIET)cd tx-client && go get $(SDS_MSG_UPD)
	$(QUIET)cd tx-client && go get $(FRAMEWORK_UPD)
	$(QUIET)cd tx-client && go mod tidy
endif

go-mod-update: go-mod-update-sds go-mod-update-tx-client

coverage:
	go test ./... -coverprofile cover.out -coverpkg=./...
	go tool cover -html cover.out -o target/cover.html
	go tool cover -func cover.out | grep total:
	rm cover.out

lint:
	golangci-lint run

.PHONY: build-linux build-mac build-docker build clean coverage lint
