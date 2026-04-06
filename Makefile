# kubectl-ondemand Makefile

BINARY_NAME := kubectl-ondemand
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: help build build-all test lint install clean release

help: ## Show this help message
	@echo "kubectl-ondemand - Karpenter on-demand node analysis"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build for current platform
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/kubectl-ondemand

build-all: ## Build for all platforms
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_linux_amd64 ./cmd/kubectl-ondemand
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_linux_arm64 ./cmd/kubectl-ondemand
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_darwin_amd64 ./cmd/kubectl-ondemand
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_darwin_arm64 ./cmd/kubectl-ondemand
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)_windows_amd64.exe ./cmd/kubectl-ondemand

test: ## Run tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

install: build ## Install to GOPATH/bin
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

release: ## Create release with goreleaser (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed"; exit 1; }
	goreleaser release --clean

release-snapshot: ## Create snapshot release (for testing)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed"; exit 1; }
	goreleaser release --snapshot --clean

tidy: ## Tidy go modules
	go mod tidy

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

check: fmt vet lint test ## Run all checks
