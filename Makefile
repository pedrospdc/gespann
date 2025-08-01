BIN_DIR := bin
TARGET := gespann
MAIN := cmd/tracker/main.go
VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build clean generate test lint fmt vet deps docker docker-build docker-run help

all: generate build

## Build the binary
build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BIN_DIR)/$(TARGET) $(MAIN)

## Generate eBPF code
generate:
	go generate ./internal/ebpf/

## Run tests
test:
	go test -v ./...

## Run linter
lint:
	golangci-lint run

## Format code
fmt:
	go fmt ./...

## Run go vet
vet:
	go vet ./...

## Install/update dependencies
deps:
	go mod download
	go mod tidy

## Clean build artifacts
clean:
	rm -rf $(BIN_DIR)
	rm -f internal/ebpf/*.o internal/ebpf/*_bpfel.go internal/ebpf/*_bpfeb.go
	docker image prune -f

## Build Docker image
docker-build:
	docker build -t gespann:$(VERSION) -t gespann:latest .

## Run Docker container
docker-run:
	docker run --rm --privileged --pid=host --network=host \
		-v /sys/fs/bpf:/sys/fs/bpf \
		-v /sys/kernel/debug:/sys/kernel/debug:ro \
		-v /sys/kernel/tracing:/sys/kernel/tracing:ro \
		-v /proc:/host/proc:ro \
		gespann:latest

## Build and run with Docker
docker: docker-build docker-run

## Development environment with docker-compose
dev-up:
	docker-compose up -d

## Stop development environment
dev-down:
	docker-compose down

## Check prerequisites
check-prereqs:
	@command -v clang >/dev/null 2>&1 || { echo "clang is required but not installed"; exit 1; }
	@command -v llvm-objcopy >/dev/null 2>&1 || { echo "llvm-objcopy is required but not installed"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "go is required but not installed"; exit 1; }

## CI pipeline
ci: check-prereqs deps fmt vet lint test build

## Show help
help:
	@echo "Available targets:"
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  %-20s %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help