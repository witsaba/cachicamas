# Makefile for github.com/cachicamas/backend/database_administrator
#
# Conventions (Go community standard):
#   - Target names mirror `go` subcommands where it makes sense.
#   - Every target is .PHONY so a file with the same name can't shadow it.
#   - Tools are installed into ./bin (LOCALBIN) so we don't pollute $GOPATH/bin.
#   - `## ` comments after a target are parsed by `make help`.
#
# Common entry points:
#   make help       # show this list
#   make install    # install Go module deps
#   make tools      # install dev tools (golangci-lint, goimports)
#   make test       # run unit tests with race detector
#   make lint       # run golangci-lint + go vet
#   make fmt        # run gofmt + goimports (writes files in place)
#   make build      # compile the server binary into ./bin/
#   make run        # run the server locally
#   make tidy       # run go mod tidy
#   make clean      # remove ./bin

# ---------- Configuration ----------------------------------------------------

BINARY_NAME   := database_administrator
LOCALBIN      := bin
MAIN_PKG      := ./src/cmd/server

# Pinned for reproducibility. Bump deliberately, not on a whim.
GOLANGCI_LINT_VERSION := v2.9.0
GOIMPORTS_VERSION     := latest
GOVULNCHECK_VERSION   := v1.1.4

# Use the project's local tool dir first so installed binaries always win.
export PATH := $(abspath $(LOCALBIN)):$(PATH)

GO          ?= go
GOFLAGS     ?= -trimpath
LDFLAGS     ?= -s -w

# Prefer local tool installations over anything on PATH.
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint
GOIMPORTS     := $(LOCALBIN)/goimports
GOVULNCHECK   := $(LOCALBIN)/govulncheck

# ---------- Helpers -----------------------------------------------------------

## help: show this help message
.PHONY: help
help:  ## Show available targets
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_\/-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf "\n"

# ---------- Dependencies -----------------------------------------------------

## install: download Go module dependencies
.PHONY: install
install:  ## Download module dependencies
	$(GO) mod download
	$(GO) mod verify

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

## vuln-check: install govulncheck and run a reachability-aware vulnerability scan
.PHONY: vuln-check
vuln-check:  ## Scan this module for known vulnerabilities (govulncheck, auto-installs if missing)
	@command -v $(GOVULNCHECK) >/dev/null 2>&1 || { \
		echo ">> installing govulncheck $(GOVULNCHECK_VERSION)"; \
		$(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION); \
		mv $$($(GO) env GOPATH)/bin/govulncheck $(LOCALBIN)/govulncheck 2>/dev/null || true; \
	}
	@echo ">> running govulncheck $(GOVULNCHECK_VERSION) against ./..."
	@$(GOVULNCHECK) ./...

## vuln-check/ci: alias of vuln-check for explicit CI workflow wiring
.PHONY: vuln-check/ci
vuln-check/ci: vuln-check  ## Same as vuln-check; alias kept explicit for future CI workflow wiring

# ---------- Build & Run -------------------------------------------------------

## build: compile the server binary into ./bin
.PHONY: build
build:  ## Compile the server binary
	@mkdir -p $(LOCALBIN)
	$(GO) build $(GOFLAGS) -ldflags='$(LDFLAGS)' -o $(LOCALBIN)/$(BINARY_NAME) $(MAIN_PKG)

## run: run the server locally
.PHONY: run
run:  ## Run the server (Ctrl-C to stop)
	$(GO) run $(MAIN_PKG)

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

# ---------- Integration tests (postgres-database-migrations, PR-A) ---------
#
# Integration tests touch a real Postgres provisioned by docker-compose.
# They are gated on INTEGRATION=1 inside the test files (see
# src/migration/postgres/driver_test.go) and booted on demand by this
# target — the operator does NOT need the full database_administrator
# container to be running, only Postgres. After the run, the Postgres
# container is stopped (data is preserved on the named volume so a
# follow-up `make test` run can re-use it).
#
# Pre-reqs:
#   - docker compose works against the project root's docker-compose.yaml
#   - the cachicamas-postgres image is already pulled (compose will
#     download on first run)
#
# Wired up by sdd-apply PR-A. Companion change:
# openspec/changes/postgres-database-migrations/.

COMPOSE_FILE ?= ../../docker-compose.yaml
COMPOSE      ?= docker compose
POSTGRES_DB  ?= cachicamas_pg
POSTGRES_USER ?= cachicamas

## test/integration: boot compose Postgres, run integration tests, stop Postgres
.PHONY: test/integration
test/integration:  ## Run integration tests (boots compose Postgres, gated on INTEGRATION=1)
	@echo ">> booting compose Postgres"
	$(COMPOSE) -f $(COMPOSE_FILE) up -d postgres
	@echo ">> waiting for Postgres to accept connections"
	$(COMPOSE) -f $(COMPOSE_FILE) exec -T postgres \
		pg_isready -U $(POSTGRES_USER) -d $(POSTGRES_DB)
	@echo ">> running integration tests (INTEGRATION=1)"
	INTEGRATION=1 $(GO) test -race -v ./...
	@echo ">> stopping compose Postgres (data volume preserved)"
	-$(COMPOSE) -f $(COMPOSE_FILE) stop postgres
	@echo ">> done"

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
