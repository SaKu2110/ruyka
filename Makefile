BINDIR:=bin
BINARY_NAME:=server
BINARIES:=bin/$(BINARY_NAME)
ROOT_PACKAGE:=$(shell go list ./)

GO_CMD:=@go
GO_BUILD:=$(GO_CMD) build
GO_INSTALL:=$(GO_CMD) install
GO_TIDY:=$(GO_CMD) mod tidy

export GO111MODULE := on

.PHONY: setup
setup:
	go install -v github.com/go-critic/go-critic/cmd/gocritic@v0.8.1
	go install -v github.com/uudashr/gocognit/cmd/gocognit@v1.0.6
	go install -v honnef.co/go/tools/cmd/staticcheck@2023.1

.PHONY: build
build: $(BINARIES)
$(BINARIES):
	$(GO_TIDY)
	$(GO_BUILD) -o $@ $(ROOT_PACKAGE);

.PHONY: clean
clean:
	rm -r $(BINDIR);

.PHONY: lint
lint:
	@gocritic check -enableAll ./...
	@gocognit .
	@staticcheck ./...
