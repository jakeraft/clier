GO      ?= go
LINTER  ?= golangci-lint

# Install location follows the same rule `go install` uses: GOBIN if set,
# otherwise GOPATH/bin. The binary is always named `clier`; channel and
# server URL are stamped via ldflags below so the same name carries
# either prod or local config — never two binaries on one machine.
GOBIN_DIR := $(shell $(GO) env GOBIN)
ifeq ($(GOBIN_DIR),)
GOBIN_DIR := $(shell $(GO) env GOPATH)/bin
endif

# Local-dev ldflags. install-local stamps channel=local + a
# commit-pinned version + localhost URLs so `clier version --json`
# reports the dev identity. The release pipeline (brew Formula etc.)
# stamps its own version + channel=release; the prod URL defaults are
# baked into the package so release builds need no URL ldflags.
COMMIT        := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LOCAL_VERSION := dev-$(COMMIT)
LOCAL_LDFLAGS := -X main.version=$(LOCAL_VERSION) \
                 -X main.channel=local \
                 -X main.commit=$(COMMIT) \
                 -X github.com/jakeraft/clier/internal/config.DefaultServerURL=http://localhost:8080 \
                 -X github.com/jakeraft/clier/internal/config.DefaultDashboardURL=http://localhost:5173

.PHONY: help check ci build install install-local _fmt _fmt_check _lint _test _build

help:
	@printf '%s\n' \
		'  make check          Local quality gate (fmt → lint → test → build)' \
		'  make ci             CI quality gate (fmt verify → lint → test → build)' \
		'  make build          Build the binary into ./clier (release defaults)' \
		'  make install        go install . — release defaults (prod URLs)' \
		'  make install-local  Build + install ./clier with channel=local' \
		'                      and localhost URLs. Use for local QA against' \
		'                      a dev server. Replaces any existing clier on PATH.'

## Public targets ─────────────────────────────────────────

check: _fmt _lint _test _build

ci: _fmt_check _lint _test _build

build:
	$(GO) build -o clier .

install:
	$(GO) install .

install-local:
	$(GO) build -ldflags '$(LOCAL_LDFLAGS)' -o $(GOBIN_DIR)/clier .
	@echo "installed clier (channel=local) at $(GOBIN_DIR)/clier"
	@echo "  → server:    http://localhost:8080"
	@echo "  → dashboard: http://localhost:5173"
	@echo "  → version:   $(LOCAL_VERSION)"

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
