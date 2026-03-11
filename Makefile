GOLANGCI_LINT ?= $(shell which golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)

.PHONY: test lint fmt vet ci clean

test: ## Run tests.
	go test ./... -count=1

test-cover: ## Run tests with coverage.
	go test ./... -coverprofile cover.out
	go tool cover -func cover.out

lint: ## Run golangci-lint.
	$(GOLANGCI_LINT) run

fmt: ## Format code.
	gofmt -s -w .

vet: ## Run go vet.
	go vet ./...

ci: lint vet test ## Run lint, vet, and test.

clean: ## Clean build artifacts.
	rm -f cover.out
