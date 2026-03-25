GO_HAVE := $(shell go version 2>/dev/null)
ifeq ($(GO_HAVE),)
  $(error 'go' not found. Install Go >= 1.21 from https://go.dev/dl/)
endif
GO_MAJ := $(shell go version | awk '{print $$3}' | sed 's/go\([0-9]*\)\..*/\1/')
GO_MIN := $(shell go version | awk '{print $$3}' | sed 's/go[0-9]*\.\([0-9]*\).*/\1/')
ifeq ($(shell test $(GO_MAJ) -gt 1 || { test $(GO_MAJ) -eq 1 && test $(GO_MIN) -ge 21; } && echo ok),)
  $(error Go >= 1.21 required (have $(shell go version)). Install from https://go.dev/dl/)
endif

VERSION := $(shell grep -E '^(const|var) Version = ' xteve.go | sed 's/.*"\(.*\)".*/\1/')
ifeq ($(VERSION),)
  $(warning WARNING: could not extract VERSION from xteve.go; using "unknown")
  VERSION := unknown
endif
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"
BUILD_DIR := build

.PHONY: build build-all clean vet test

build:
	mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/xteve .

build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/xteve_linux_amd64  .
	GOOS=linux GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/xteve_linux_arm64  .
	GOOS=linux GOARCH=arm    GOARM=7 go build $(LDFLAGS) -o $(BUILD_DIR)/xteve_linux_arm .

clean:
	rm -rf $(BUILD_DIR)

vet:
	go vet ./...

test:
	go test ./...
