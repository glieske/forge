.PHONY: dev test lint fmt fmt-check build build-all examples-check ci package-release

GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod
GOLANGCI_LINT ?= golangci-lint
export GOCACHE
export GOMODCACHE

dev:
	./scripts/dev.sh

test:
	go test ./...

lint:
	go vet ./...
	$(GOLANGCI_LINT) run ./... --timeout=5m

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l cmd internal tools examples/plugins)"

build:
	go build -o bin/forge ./cmd/forge

build-all:
	./scripts/build-all.sh

examples-check:
	sh -n examples/plugins/connect/package.sh
	cd examples/plugins/connect && go mod tidy
	cd examples/plugins/connect && go vet ./...
	cd examples/plugins/connect && go test ./...
	cd examples/plugins/connect && go build -o ../../../bin/forge-connect-example .

ci: fmt-check lint test build build-all examples-check

package-release:
	./scripts/package-release.sh
