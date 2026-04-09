GO ?= go
GOLANGCI_LINT ?= golangci-lint
GORELEASER ?= goreleaser

.PHONY: help bootstrap check fmt lint test build install run ci release

help:
	@printf '%s\n' \
		'First time:' \
		'  make bootstrap  Install local developer tools and clier itself' \
		'' \
		'Daily workflow:' \
		'  make fmt        Format Go sources' \
		'  make check      Run lint, test, and build locally' \
		'  make run        Run clier with go run .' \
		'' \
		'CI and release:' \
		'  make ci         Run the CI command set' \
		'  make release    Build a snapshot release with goreleaser' \
		'' \
		'Individual commands:' \
		'  make lint       Run golangci-lint' \
		'  make test       Run go test ./...' \
		'  make build      Run go build ./...' \
		'  make install    Install clier with go install .'

bootstrap: install

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

release:
	$(GORELEASER) release --snapshot --clean
