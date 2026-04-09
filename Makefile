GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: help check fmt lint test build install run ci

help:
	@printf '%s\n' \
		'Dev quality:' \
		'  make fmt         Format Go sources' \
		'  make lint        Run golangci-lint' \
		'  make test        Run go test ./...' \
		'  make build       Run go build ./...' \
		'  make check       Run lint, test, and build locally' \
		'' \
		'Local execution:' \
		'  make run         Run clier with go run .' \
		'  make install     Install clier with go install .' \
		'' \
		'CI:' \
		'  make ci          Run the CI command set'

check: lint test build

fmt:
	@find . -type f -name '*.go' -print0 | xargs -0 gofmt -w

lint:
	$(GOLANGCI_LINT) run

test:
	$(GO) test ./...

build:
	$(GO) build ./...

install:
	$(GO) install .

run:
	$(GO) run .

ci: check
