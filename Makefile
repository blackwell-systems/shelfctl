.PHONY: build clean test test-coverage install lint fmt vet ci-local

BINARY  := bin/shelfctl
MAIN    := ./cmd/shelfctl
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)

build:
	@mkdir -p bin
	GOWORK=off go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

test:
	go test -v ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

install:
	GOWORK=off go install -ldflags "$(LDFLAGS)" $(MAIN)

clean:
	rm -rf bin/ coverage.out coverage.html

lint:
	golangci-lint run --timeout=5m

fmt:
	gofmt -w .

fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Go files must be formatted with gofmt:"; \
		gofmt -l .; \
		exit 1; \
	fi

vet:
	go vet ./...

# Run all CI checks locally
ci-local: fmt-check vet test build
	@echo "✓ All CI checks passed"

.DEFAULT_GOAL := build
