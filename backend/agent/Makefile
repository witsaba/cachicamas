# Makefile for github.com/cachicamas/backend/agent
#
# Derived from backend/database_administrator/Makefile. The copy is deliberate:
# a shared build config would be a fourth thing to own and would couple two
# modules that ADR 0005 requires to stay uncoupled. ADR 0005 records the drift
# risk this creates.
#
# Two targets are load-bearing for the evidence gate of every milestone in
# docs 0002, 0003 and 0004. Their bodies are pinned so a later divergence from
# database_administrator is visible rather than silent:
#
#   test  -> go test -race -v ./...
#   lint  -> go vet, then golangci-lint at the pinned version below
#
# Common entry points:
#   make help       # show this list
#   make tools      # install dev tools (golangci-lint, goimports)
#   make test       # run unit tests with race detector
#   make lint       # run golangci-lint + go vet
#   make fmt        # run gofmt + goimports (writes files in place)
#   make build      # compile every package
#   make tidy       # run go mod tidy
#   make clean      # remove ./bin

# ---------- Configuration ----------------------------------------------------

LOCALBIN      := bin

# Pinned for reproducibility. Bump deliberately, not on a whim.
# Must match backend/database_administrator/Makefile.
GOLANGCI_LINT_VERSION := v2.9.0
GOIMPORTS_VERSION     := latest

# Use the project's local tool dir first so installed binaries always win.
export PATH := $(abspath $(LOCALBIN)):$(PATH)

GO          ?= go
GOFLAGS     ?= -trimpath

# Prefer local tool installations over anything on PATH.
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint
GOIMPORTS     := $(LOCALBIN)/goimports

# ---------- Helpers -----------------------------------------------------------

## help: show this help message
.PHONY: help
help:  ## Show available targets
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_\/-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf "\n"

# ---------- Dependencies -----------------------------------------------------
#
# There is no `install` target on purpose. This module carries ZERO requires
# and has nothing to download. It stays that way until a milestone arrives with
# an ADR to add a dependency: AI-24 selects a transport (its own ADR gate), and
# AI-37 adds the OpenTelemetry API (pre-authorised by ADR 0005 section D3, and
# by nothing else).

## tidy: clean go.mod / go.sum
.PHONY: tidy
tidy:  ## Run `go mod tidy`
	$(GO) mod tidy

## tools: install development tools into ./bin
.PHONY: tools
tools:  ## Install golangci-lint + goimports into ./bin
	@mkdir -p $(LOCALBIN)
	@echo ">> installing golangci-lint $(GOLANGCI_LINT_VERSION)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
		| sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)
	@echo ">> installing goimports"
	@$(GO) install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
	@mv $$($(GO) env GOPATH)/bin/goimports $(LOCALBIN)/goimports 2>/dev/null || true

# ---------- Build -------------------------------------------------------------
#
# `build` compiles every package rather than a named binary: this module has no
# `main` package yet. The composition root arrives with doc 0004
# (src/cmd/cachicamas). The target name stays stable for that day.

## build: compile every package
.PHONY: build
build:  ## Compile all packages
	$(GO) build $(GOFLAGS) ./...

# ---------- Tests -------------------------------------------------------------

## test: run all tests with race detector
.PHONY: test
test:  ## Run unit tests (`go test -race -v ./...`)
	$(GO) test -race -v ./...

## test/cover: run tests with coverage report
.PHONY: test/cover
test/cover:  ## Run tests and write coverage.out
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	@$(GO) tool cover -func=coverage.out | tail -1

# ---------- Quality -----------------------------------------------------------

## fmt: run gofmt and goimports (writes files in place)
.PHONY: fmt
fmt:  ## Format code with gofmt + goimports
	$(GO) fmt ./...
	@if [ -x $(GOIMPORTS) ]; then $(GOIMPORTS) -w .; else echo "goimports not installed; run 'make tools'"; fi

## vet: run go vet
.PHONY: vet
vet:  ## Run `go vet ./...`
	$(GO) vet ./...

## lint: run golangci-lint (requires `make tools` first)
.PHONY: lint
lint: vet  ## Run golangci-lint (auto-installs if missing)
	@if [ ! -x $(GOLANGCI_LINT) ]; then \
		echo ">> golangci-lint not found, installing..."; \
		$(MAKE) tools; \
	fi
	$(GOLANGCI_LINT) run --config=.golangci.yml ./...

# ---------- Housekeeping ------------------------------------------------------

## clean: remove build artifacts
.PHONY: clean
clean:  ## Remove ./bin and coverage files
	rm -rf $(LOCALBIN) coverage.out coverage.html

.PHONY: all
all: tidy fmt vet lint test build  ## Run the full local CI pipeline
