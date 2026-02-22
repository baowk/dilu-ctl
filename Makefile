# Makefile for dilu-ctl

BINARY_NAME=dilu-ctl
VERSION ?= dev

# Build for current platform
build:
	go build -o ${BINARY_NAME} .

# Install to GOPATH
install:
	go install .

# Cross compilation
build-all: build-linux build-windows build-macos

build-linux:
	GOOS=linux GOARCH=amd64 go build -o ${BINARY_NAME}-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o ${BINARY_NAME}-linux-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o ${BINARY_NAME}-windows-amd64.exe .

build-macos:
	GOOS=darwin GOARCH=amd64 go build -o ${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o ${BINARY_NAME}-darwin-arm64 .

# Clean build artifacts
clean:
	rm -f ${BINARY_NAME} ${BINARY_NAME}-*

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Check code quality
check: fmt vet test

# Release preparation
release: clean check build-all
	@echo "Release v${VERSION} ready"
	@ls -la ${BINARY_NAME}*

.PHONY: build install build-all build-linux build-windows build-macos clean test test-cover fmt vet check release