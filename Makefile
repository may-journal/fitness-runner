# Makefile — command index for fitness-runner.
# The Go module lives in ./go, so every target runs there.
# Commands mirror .github/workflows (ci.yml / release.yml) so `make` matches CI.

GO_DIR := go
# Extra args for `make run`, e.g. `make run ARGS="--help"`.
ARGS ?=

.PHONY: help build install test test-race cover vet fmt fmt-check lint check run tidy clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-11s\033[0m %s\n", $$1, $$2}'

build: ## Compile all cmd/ binaries into go/bin
	cd $(GO_DIR) && mkdir -p bin && go build -o bin ./cmd/...

install: ## Install all cmd/ binaries to GOBIN (on your PATH)
	cd $(GO_DIR) && go install ./cmd/...

test: ## Run all tests
	cd $(GO_DIR) && go test ./...

test-race: ## Run all tests with the race detector
	cd $(GO_DIR) && go test -race ./...

cover: ## Run tests with coverage summary
	cd $(GO_DIR) && go test -cover ./...

vet: ## Run go vet
	cd $(GO_DIR) && go vet ./...

fmt: ## Format all Go sources in place
	gofmt -w $(GO_DIR)

fmt-check: ## Fail if any Go source is unformatted (CI check)
	@test -z "$$(gofmt -l $(GO_DIR))" || { echo "Unformatted files:"; gofmt -l $(GO_DIR); exit 1; }

lint: fmt-check vet ## Run format check + go vet

check: lint test ## Run everything CI runs (lint + tests)

run: build ## Run the fitness suite on this repo, e.g. make run ARGS="--check=cspell"
	./$(GO_DIR)/bin/fitness $(ARGS)

tidy: ## Tidy go.mod / go.sum
	cd $(GO_DIR) && go mod tidy

clean: ## Remove compiled binaries (go/bin)
	rm -rf $(GO_DIR)/bin

.DEFAULT_GOAL := help
