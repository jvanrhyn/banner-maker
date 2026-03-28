# ─── banner-maker Makefile ────────────────────────────────────────────────────

BINARY     := banner-maker
MODULE     := $(shell go list -m)
_GOBIN      := $(shell go env GOBIN)
INSTALL_DIR := $(if $(_GOBIN),$(_GOBIN),$(shell go env GOPATH)/bin)

# ─── Version (SemVer 2.0) ─────────────────────────────────────────────────────
# Base: MAJOR.MINOR.PATCH read from VERSION file
# Build metadata: +{commit_count}.{short_sha}[.dirty]
BASE_VERSION := $(shell cat VERSION | tr -d '[:space:]')
COMMIT_COUNT := $(shell git rev-list --count HEAD 2>/dev/null || echo "0")
SHORT_SHA    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DIRTY        := $(shell git diff --quiet 2>/dev/null && git diff --cached --quiet 2>/dev/null || echo ".dirty")
VERSION      := $(BASE_VERSION)+$(COMMIT_COUNT).$(SHORT_SHA)$(DIRTY)
LDFLAGS      := -ldflags "-X main.version=$(VERSION)"

# ─── Targets ──────────────────────────────────────────────────────────────────

.DEFAULT_GOAL := help

.PHONY: help build install uninstall test coverage lint clean version

help: ## Show this help
	@echo "banner-maker $(VERSION)"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

build: ## Build binary to ./banner-maker
	go build $(LDFLAGS) -o $(BINARY) .

install: ## Install binary to GOPATH/bin
	go install $(LDFLAGS) .
	@echo "Installed $(BINARY) $(VERSION) → $(INSTALL_DIR)/$(BINARY)"

uninstall: ## Remove binary from GOPATH/bin
	@rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Uninstalled $(INSTALL_DIR)/$(BINARY)"

test: ## Run all tests
	go test ./...

coverage: ## Run tests with HTML coverage report (opens coverage.html)
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run go vet and staticcheck (if available)
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed — skipping (go install honnef.co/go/tools/cmd/staticcheck@latest)"; \
	fi

clean: ## Remove build artifacts
	@rm -f $(BINARY) coverage.out coverage.html debug.log
	@echo "Cleaned"

version: ## Print current version string
	@echo $(VERSION)
