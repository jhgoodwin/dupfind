SHELL := /bin/sh
.SHELLFLAGS := -eu -c

GOCACHE := $(CURDIR)/.gocache
COVERAGE_MIN := 60
COMPLEXITY_MAX := 30
BIN_NAME := dupfind
BIN_DIR := bin
INSTALL_DIR := $(HOME)/.local/bin

export GOCACHE

GOPATH := $(shell go env GOPATH)
PATH := $(GOPATH)/bin:$(PATH)
export GOPATH PATH

.PHONY: build install setup install-dev-tools install-hooks pre-commit go-cache tidy fmt tidy-fmt-check vet cyclo test test-slow coverage gtags dup-check

setup: install-dev-tools install-hooks go-cache

build: go-cache
	@echo "build: $(BIN_NAME)"
	@go build -o $(BIN_DIR)/$(BIN_NAME) ./internal/dupfind

install: build
	@echo "install: $(BIN_NAME) -> $(INSTALL_DIR)"
	@mkdir -p $(INSTALL_DIR)
	@cp $(BIN_DIR)/$(BIN_NAME) $(INSTALL_DIR)/$(BIN_NAME)

install-dev-tools:
	@echo "install: developer tools"
	@if ! command -v gocyclo >/dev/null 2>&1; then \
		go install github.com/fzipp/gocyclo/cmd/gocyclo@latest; \
	fi
	@if ! command -v gtags >/dev/null 2>&1; then \
		if ! command -v brew >/dev/null 2>&1; then \
			echo "install: gtags is required; install GNU Global or install Homebrew and rerun make setup" >&2; \
			exit 1; \
		fi; \
		brew install global universal-ctags; \
	fi

install-hooks:
	@echo "install: git hooks"
	@git config core.hooksPath .githooks

pre-commit: tidy-fmt-check vet cyclo coverage gtags dup-check
	@echo "pre-commit: ok"

go-cache:
	@mkdir -p "$(GOCACHE)"

tidy:
	@echo "go mod tidy"
	@go mod tidy

fmt:
	@echo "gofmt"
	@go_files=$$(git ls-files '*.go'); \
	if [ -n "$$go_files" ]; then \
		gofmt -w $$go_files; \
	fi

tidy-fmt-check: go-cache
	@dirty_before=$$(mktemp "$${TMPDIR:-/tmp}/dupfind-pre-commit-before.XXXXXX"); \
	dirty_after=$$(mktemp "$${TMPDIR:-/tmp}/dupfind-pre-commit-after.XXXXXX"); \
	trap 'rm -f "$$dirty_before" "$$dirty_after"' EXIT INT TERM; \
	git diff -- go.mod go.sum '*.go' >"$$dirty_before"; \
	echo "pre-commit: go mod tidy"; \
	go mod tidy; \
	echo "pre-commit: gofmt"; \
	go_files=$$(git ls-files '*.go'); \
	if [ -n "$$go_files" ]; then \
		gofmt -w $$go_files; \
	fi; \
	git diff -- go.mod go.sum '*.go' >"$$dirty_after"; \
	if ! cmp -s "$$dirty_before" "$$dirty_after"; then \
		echo "pre-commit: gofmt or go mod tidy changed files; stage those changes and commit again" >&2; \
		exit 1; \
	fi

vet: go-cache
	@echo "pre-commit: go vet"
	@go vet ./...

cyclo:
	@echo "pre-commit: gocyclo"
	@if ! command -v gocyclo >/dev/null 2>&1; then \
		echo "pre-commit: gocyclo is required; install with: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest" >&2; \
		exit 1; \
	fi
	@complexity=$$(gocyclo -over $(COMPLEXITY_MAX) .); \
	if [ -n "$$complexity" ]; then \
		echo "$$complexity" >&2; \
		echo "pre-commit: complexity exceeds threshold $(COMPLEXITY_MAX)" >&2; \
		exit 1; \
	fi

test: go-cache
	@echo "go test (fast; use 'make test-slow' for full suite)"
	@go test ./...

test-slow: go-cache
	@echo "go test -tags=slow (full suite)"
	@go test -tags=slow ./...

coverage: go-cache
	@echo "pre-commit: go test -tags=slow"
	@coverage_file=$$(mktemp "$${TMPDIR:-/tmp}/dupfind-coverage.XXXXXX"); \
	trap 'rm -f "$$coverage_file"' EXIT INT TERM; \
	go test -tags=slow ./... -coverprofile="$$coverage_file"; \
	coverage=$$(go tool cover -func="$$coverage_file" | awk '/^total:/ { sub(/%$$/, "", $$3); print $$3 }'); \
	awk -v coverage="$$coverage" 'BEGIN { if (coverage < $(COVERAGE_MIN)) exit 1 }' || { \
		echo "pre-commit: total coverage $$coverage% is below required $(COVERAGE_MIN)%" >&2; \
		exit 1; \
	}

gtags:
	@echo "pre-commit: gtags"
	@if ! command -v gtags >/dev/null 2>&1; then \
		echo "pre-commit: gtags is required to rebuild GNU Global tags" >&2; \
		exit 1; \
	fi
	@gtags --gtagslabel=new-ctags

dup-check: build
	@echo "dup-check:"
	@$(BIN_DIR)/$(BIN_NAME) -root internal/
