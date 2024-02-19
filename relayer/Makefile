BUILDDIR ?= $(CURDIR)/target

SDS_GIT_REVISION := $(shell git log --pretty=format:"%h" --abbrev-commit -1)
SDS_MSG_UPD := github.com/stratosnet/sds/sds-msg@$(SDS_GIT_REVISION)
FRAMEWORK_UPD := github.com/stratosnet/sds/framework@$(SDS_GIT_REVISION)
TX_CLIENT_UPD := github.com/stratosnet/sds/tx-client@$(SDS_GIT_REVISION)

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

go-mod-update:
	go get $(SDS_MSG_UPD)
	go get $(FRAMEWORK_UPD)
	go get $(TX_CLIENT_UPD)
	go mod tidy
