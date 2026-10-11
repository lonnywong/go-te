SHELL := /bin/sh

.DEFAULT_GOAL := help

# Disposable storage: explicit override; CI runner/inherited tmp; local workspace.
INHERITED_TMPDIR := $(TMPDIR)
CI_MODE := $(filter-out 0 false,$(strip $(CI)))
GENERIC_TMP_BASE := $(if $(strip $(INHERITED_TMPDIR)),$(INHERITED_TMPDIR),/tmp)
ifeq ($(strip $(WORKSPACE_TMP_BASE)),)
ifneq ($(CI_MODE),)
TMP_BASE := $(if $(strip $(RUNNER_TEMP)),$(RUNNER_TEMP),$(GENERIC_TMP_BASE))
else
TMP_BASE := $(shell if test -d /workspace/tmp && test -w /workspace/tmp; then printf /workspace/tmp; else printf '%s' '$(GENERIC_TMP_BASE)'; fi)
endif
else
TMP_BASE := $(WORKSPACE_TMP_BASE)
endif
PROJECT_TMP := $(abspath $(TMP_BASE)/go-te)
export TMPDIR := $(PROJECT_TMP)/runs
export GOCACHE := $(PROJECT_TMP)/cache/go-build
export GOMODCACHE := $(PROJECT_TMP)/cache/go-mod
export GOTMPDIR := $(PROJECT_TMP)/build/go-tmp
export GOTOOLCHAIN := local
COVERAGE_FILE := $(PROJECT_TMP)/tests/coverage.out

.PHONY: dirs
dirs:
	@mkdir -p "$(TMPDIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOTMPDIR)" "$(PROJECT_TMP)/tests" "$(PROJECT_TMP)/logs" "$(PROJECT_TMP)/runs/profiles" "$(PROJECT_TMP)/build/analysis"

# Generic Makefile template.
# Projects can extend/override these targets.

IMAGE ?= $(notdir $(CURDIR))
TAG ?= latest
FULL_IMAGE := $(IMAGE):$(TAG)
REGISTRY ?= ghcr.io
GHCR_OWNER ?= $(shell whoami)
GHCR_IMAGE := $(REGISTRY)/$(GHCR_OWNER)/$(IMAGE):$(TAG)

.PHONY: help
help: ## Show targets
	@grep -E '^[a-zA-Z0-9_.-]+:.*?##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-18s %s\\n", $$1, $$2}'

# =============================================================================
# Docker
# =============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(FULL_IMAGE) .

.PHONY: dual-tag
dual-tag: docker-build ## Tag image as ghcr.io/<user>/<image>:<tag>
	docker tag $(FULL_IMAGE) $(GHCR_IMAGE)

.PHONY: tag-ghcr
tag-ghcr: dual-tag ## Convenience alias for dual-tag

# =============================================================================
# Dependency install
# =============================================================================

.PHONY: install
install: ## Install project dependencies
	@$(MAKE) deps

.PHONY: install-dev
install-dev: ## Install dev dependencies
	@$(MAKE) deps

.PHONY: deps
deps: dirs ## Install Go toolchain dependencies
	@command -v go >/dev/null 2>&1 || { echo "Go is required"; exit 1; }
	go mod download
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi

# =============================================================================
# Quality
# =============================================================================

.PHONY: vet
vet: dirs ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run --timeout=5m

.PHONY: security
security: ## Run gosec
	gosec ./...

.PHONY: test
test: dirs ## Run tests, race detection and scoped coverage
	go test -v -race -coverprofile="$(COVERAGE_FILE)" ./...

.PHONY: coverage
coverage: dirs ## Run tests with scoped coverage
	go test -v -coverprofile="$(COVERAGE_FILE)" ./...

.PHONY: check
check: ## Run standard validation pipeline
	@$(MAKE) vet && $(MAKE) lint

# =============================================================================
# Release
# =============================================================================

.PHONY: bump-patch
bump-patch: ## Bump patch version in VERSION and create git tag
	@if [ ! -f VERSION ]; then echo "VERSION file not found"; exit 1; fi
	@OLD=$$(cat VERSION); \
	MAJOR=$$(echo $$OLD | cut -d. -f1); \
	MINOR=$$(echo $$OLD | cut -d. -f2); \
	PATCH=$$(echo $$OLD | cut -d. -f3); \
	NEW="$$MAJOR.$$MINOR.$$((PATCH + 1))"; \
	echo $$NEW > VERSION; \
	git add VERSION; \
	git commit -m "Bump version to $$NEW"; \
	git tag "v$$NEW"; \
	echo "Bumped version: $$OLD -> $$NEW (tagged v$$NEW)"

.PHONY: push
push: ## Push commits and current tag to origin
	@TAG=$$(git describe --tags --exact-match 2>/dev/null); \
	git push origin main; \
	if [ -n "$$TAG" ]; then \
		echo "Pushing tag $$TAG..."; \
		git push origin "$$TAG"; \
	else \
		echo "No tag on current commit"; \
	fi

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean
clean: ## Remove owned test/build outputs; retain shared caches
	@test "$(PROJECT_TMP)" != / && test "$$(basename "$(PROJECT_TMP)")" = go-te
	rm -rf "$(PROJECT_TMP)/tests" "$(PROJECT_TMP)/build"

.PHONY: benchmark profile
benchmark: dirs ## Run allocation benchmarks without profiling
	go test ./pkg/te -run '^$$' -bench . -benchmem -count=5
profile: dirs ## Profile the complete tests of every test-bearing package
	@set -e; for pkg in $$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...); do \
		name=$$(printf '%s' "$$pkg" | tr '/.' '--'); \
		go test "$$pkg" -count=1 -cpuprofile="$(PROJECT_TMP)/runs/profiles/$$name.cpu" -memprofile="$(PROJECT_TMP)/runs/profiles/$$name.mem" -memprofilerate=1 -o "$(PROJECT_TMP)/build/analysis/$$name.test"; \
	done
