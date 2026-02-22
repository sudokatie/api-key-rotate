VERSION ?= 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

.PHONY: build test lint clean

build:
	go build $(LDFLAGS) -o bin/api-key-rotate ./cmd/api-key-rotate

test:
	go test -v -race -cover ./...

lint:
	go vet ./...
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed"

clean:
	rm -rf bin/
	go clean -testcache

install: build
	cp bin/api-key-rotate /usr/local/bin/

.DEFAULT_GOAL := build
