.PHONY: build install uninstall test clean dev fmt vet lint check snapshot release docs

BINARY_NAME := portman
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X github.com/tasnimzotder/portman/internal/cli.Version=$(VERSION) -X github.com/tasnimzotder/portman/internal/cli.Commit=$(COMMIT) -X github.com/tasnimzotder/portman/internal/cli.Date=$(DATE)

## Build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/portman

install: build
	cp bin/$(BINARY_NAME) /usr/local/bin/

uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)

## Development
dev: build
	./bin/$(BINARY_NAME)

test:
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

## Release
check:
	goreleaser check

snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

## Docs
docs:
	uvx --with mkdocs-material mkdocs serve

## Changelog
changelog:
	git-cliff --output CHANGELOG.md

changelog-preview:
	git-cliff --unreleased

changelog-update:
	git-cliff --unreleased --prepend CHANGELOG.md

## Cleanup
clean:
	rm -rf bin/ dist/
