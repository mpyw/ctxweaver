.PHONY: all build test lint fmt clean install

# Default target
all: lint test build

# Build the CLI binary
build:
	go build -v -o bin/ctxweaver ./cmd/ctxweaver

# Install the CLI binary
install:
	go install ./cmd/ctxweaver

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	goimports -w -local github.com/mpyw/ctxweaver .
	gofmt -w -s .

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Verify go.mod is tidy
tidy:
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod is not tidy" && exit 1)
