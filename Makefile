GO      ?= go
LINTER  ?= golangci-lint

.PHONY: help check build install _fmt _lint _test _build

help:
	@printf '%s\n' \
		'  make check     Local quality gate (fmt → lint → test → build)' \
		'  make build     Build production binary' \
		'  make install   Install clier locally'

## Public targets ─────────────────────────────────────────

check: _fmt _lint _test _build

build:
	$(GO) build -o clier .

install:
	$(GO) install .

## Internal targets ───────────────────────────────────────

_fmt:
	@gofmt -w .

_lint:
	$(LINTER) run

_test:
	$(GO) test ./...

_build:
	$(GO) build ./...
