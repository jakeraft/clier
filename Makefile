GO      ?= go
LINTER  ?= golangci-lint

.PHONY: help check ci build install _fmt _fmt_check _lint _test _build

help:
	@printf '%s\n' \
		'  make check     Local quality gate (fmt → lint → test → build)' \
		'  make ci        CI quality gate (fmt verify → lint → test → build)' \
		'  make build     Build production binary' \
		'  make install   Install clier locally'

## Public targets ─────────────────────────────────────────

check: _fmt _lint _test _build

ci: _fmt_check _lint _test _build

build:
	$(GO) build -o clier .

install:
	$(GO) install .

## Internal targets ───────────────────────────────────────

_fmt:
	@gofmt -w .

_fmt_check:
	@diff=$$(gofmt -l .); if [ -n "$$diff" ]; then \
		echo "gofmt needs to be run on:"; echo "$$diff"; exit 1; \
	fi

_lint:
	$(LINTER) run

_test:
	$(GO) test ./...

_build:
	$(GO) build ./...
