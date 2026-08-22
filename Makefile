# SPDX-License-Identifier: GPL-3.0-or-later
# Copyright (C) 2026 The Granite Authors

GO      ?= go
GOBIN   := $(shell $(GO) env GOPATH)/bin
PKGS    := ./...

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-z-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[1m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: tools
tools: ## Install the quality tooling into $(GOBIN)
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install honnef.co/go/tools/cmd/staticcheck@latest
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$(GO) install github.com/google/go-licenses@latest
	$(GO) install github.com/fe3dback/go-arch-lint@latest

.PHONY: hooks
hooks: ## Install the repository git hooks (DCO sign-off check)
	git config core.hooksPath .githooks
	@echo "git hooks installed from .githooks/"

.PHONY: build
build: ## Build all binaries
	$(GO) build $(PKGS)

.PHONY: test
test: ## Run tests with the race detector (QA-005)
	$(GO) test -race $(PKGS)

.PHONY: fmt
fmt: ## Rewrite files with gofumpt
	$(GOBIN)/gofumpt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofumpt-clean (QA-011)
	@out=$$($(GOBIN)/gofumpt -l .); \
	if [ -n "$$out" ]; then echo "not gofumpt-clean:"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## Run go vet
	$(GO) vet $(PKGS)

.PHONY: lint
lint: ## Run staticcheck (QA-011)
	$(GOBIN)/staticcheck $(PKGS)

.PHONY: sec
sec: ## Run gosec (QA-011)
	$(GOBIN)/gosec -quiet $(PKGS)

.PHONY: vuln
vuln: ## Run govulncheck (NFR-SEC-009)
	$(GOBIN)/govulncheck $(PKGS)

.PHONY: licenses
licenses: ## Regenerate THIRD-PARTY-LICENSES.md (LEG-006)
	./scripts/gen-third-party-licenses.sh

.PHONY: licenses-check
licenses-check: ## Fail if THIRD-PARTY-LICENSES.md is stale or a license is incompatible (LEG-005)
	@./scripts/gen-third-party-licenses.sh --check

.PHONY: spdx-check
spdx-check: ## Fail if any Go file lacks the SPDX header
	@missing=$$(find . -name '*.go' -not -path './.git/*' \
		-exec grep -L 'SPDX-License-Identifier: GPL-3.0-or-later' {} +); \
	if [ -n "$$missing" ]; then echo "missing SPDX header:"; echo "$$missing"; exit 1; fi

.PHONY: check
check: build test fmt-check vet lint sec vuln licenses-check spdx-check ## Run every gate a pull request must pass
	@echo "all checks passed"

.PHONY: clean
clean: ## Remove build artifacts
	$(GO) clean
	rm -rf dist/
