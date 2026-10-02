.PHONY: all build test lint clean install fmt vet validate

# Variables
BINARY_NAME := dtiam
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/timstewart-dynatrace/dtiam/v3/pkg/version.Version=$(VERSION) \
                     -X github.com/timstewart-dynatrace/dtiam/v3/pkg/version.Commit=$(COMMIT) \
                     -X github.com/timstewart-dynatrace/dtiam/v3/pkg/version.Date=$(DATE)"

# Default target
all: fmt vet lint test build

# Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/dtiam

# Build for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/dtiam
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/dtiam
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/dtiam
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/dtiam
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/dtiam

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install the binary
install:
	go install $(LDFLAGS) ./cmd/dtiam

# Download dependencies
deps:
	go mod download
	go mod tidy

# Update dependencies
update-deps:
	go get -u ./...
	go mod tidy

# Run the CLI (for development)
run:
	go run ./cmd/dtiam $(ARGS)

# Generate mocks (if using mockery)
mocks:
	@which mockery > /dev/null || (echo "Installing mockery..." && go install github.com/vektra/mockery/v2@latest)
	mockery --all --output ./internal/mocks

# Run comprehensive validation
validate: build
	./scripts/validate.sh

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Format, vet, lint, test, and build"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  validate     - Run comprehensive validation script"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install the binary"
	@echo "  deps         - Download dependencies"
	@echo "  update-deps  - Update dependencies"
	@echo "  run          - Run the CLI (use ARGS='...' to pass arguments)"
	@echo "  help         - Show this help"
