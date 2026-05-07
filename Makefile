GO      ?= go
LINTER  ?= golangci-lint

# Install location follows the same rule `go install` uses: GOBIN if set,
# otherwise GOPATH/bin. The binary is always named `clier`; channel and
# server URL are stamped via ldflags below so the same name carries
# either prod or dev config — never two binaries on one machine.
#
# Channel vocabulary: prod (default) / dev. The unmarked path —
# `go install`, `go build`, brew install — is always prod. Only
# `make install-dev` flips the channel to dev and points the URLs at
# localhost. main.channel defaults to "prod" too, so a stamped binary
# and an unstamped one report the same channel for the same install
# path.
GOBIN_DIR := $(shell $(GO) env GOBIN)
ifeq ($(GOBIN_DIR),)
GOBIN_DIR := $(shell $(GO) env GOPATH)/bin
endif

# install-dev ldflags. Stamps channel=dev + a commit-pinned version +
# localhost URLs so `clier version` reports the dev identity. Prod is
# the package default — `go install` / brew install / GoReleaser need
# no URL ldflags.
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DEV_VERSION := dev-$(COMMIT)
DEV_LDFLAGS := -X main.version=$(DEV_VERSION) \
               -X main.channel=dev \
               -X main.commit=$(COMMIT) \
               -X github.com/jakeraft/clier/internal/config.DefaultServerURL=http://localhost:8080 \
               -X github.com/jakeraft/clier/internal/config.DefaultDashboardURL=http://localhost:5173

.PHONY: help check ci build install install-dev _fmt _fmt_check _lint _test _build

help:
	@printf '%s\n' \
		'  make check        Local quality gate (fmt → lint → test → build)' \
		'  make ci           CI quality gate (fmt verify → lint → test → build)' \
		'  make build        Build the binary into ./clier (prod defaults)' \
		'  make install      go install . — prod defaults' \
		'  make install-dev  Build + install ./clier with channel=dev' \
		'                    and localhost URLs. Use for local QA against' \
		'                    a dev server. Replaces any existing clier on PATH.'

## Public targets ─────────────────────────────────────────

check: _fmt _lint _test _build

ci: _fmt_check _lint _test _build

build:
	$(GO) build -o clier .

install:
	$(GO) install .

install-dev:
	$(GO) build -ldflags '$(DEV_LDFLAGS)' -o $(GOBIN_DIR)/clier .
	@echo "installed clier (channel=dev) at $(GOBIN_DIR)/clier"
	@echo "  → server:    http://localhost:8080"
	@echo "  → dashboard: http://localhost:5173"
	@echo "  → version:   $(DEV_VERSION)"

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
