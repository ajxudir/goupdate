.PHONY: init build test test-ci test-unit test-e2e coverage coverage-func coverage-html vet fmt lint check clean install

# Version information (can be overridden: make build VERSION=1.0.0)
# Use exact git tag if current commit is tagged, otherwise "dev"
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build target (defaults to current platform, can be overridden for cross-compilation)
BUILD_OS ?= $(shell go env GOOS)
BUILD_ARCH ?= $(shell go env GOARCH)

LDFLAGS := -X github.com/user/goupdate/cmd.Version=$(VERSION) \
           -X github.com/user/goupdate/cmd.BuildTime=$(BUILD_TIME) \
           -X github.com/user/goupdate/cmd.GitCommit=$(GIT_COMMIT) \
           -X github.com/user/goupdate/cmd.BuildOS=$(BUILD_OS) \
           -X github.com/user/goupdate/cmd.BuildArch=$(BUILD_ARCH)

# Initialize and download dependencies
init:
	go mod tidy
	go mod download

# Build binary with version information
build:
	go build -ldflags="$(LDFLAGS)" -o goupdate main.go

# Build without version stamping (faster for development)
build-dev:
	go build -o goupdate main.go

# Run all package tests recursively with race detector. The ./... pattern tells
# Go to include every package beneath the current module root, regardless of
# where this Makefile lives.
test:
	go test -race -v ./...

# Run tests quietly (no verbose output) - for CI pipelines
# Only shows output on failure
test-ci:
	go test -race ./...

# Run unit tests only
test-unit:
	go test -race -v ./cmd ./pkg

# Run e2e tests only
test-e2e:
	go test -race -v ./... -run EndToEnd

# Generate coverage profile
#
# The race detector dramatically slows the first run because it must rebuild
# the standard library with race instrumentation. That long rebuild was being
# misinterpreted as a hung or stopped process when running "make coverage" in
# interactive shells. We keep the race-enabled test target for full runs, but
# drop the flag here so coverage completes quickly and reliably.
#
# Note: We explicitly list packages to exclude the main package from coverage.
# The main.go file only contains the entry point (main function) which cannot
# be tested via go test - this is standard Go best practice. All testable logic
# is in the cmd and pkg packages.
coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./cmd/... ./pkg/...
	@echo "Coverage profile generated: coverage.out"

# Display coverage in terminal
coverage-func: coverage
	go tool cover -func=coverage.out

# Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run go vet for static analysis
vet:
	go vet ./...

# Format code using gofmt
fmt:
	gofmt -s -w .
	@echo "Code formatted"

# Run Go linters (requires golangci-lint: https://golangci-lint.run/usage/install/)
lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

# Run all checks (vet, fmt check, tests)
check:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Please run 'make fmt'" && exit 1)
	@echo "Running tests..."
	@go test -race ./...
	@echo "All checks passed!"

# Clean build artifacts
clean:
	rm -f goupdate coverage.out coverage.html
	go clean -cache -testcache

# Install binary
install: build
	go install
