.PHONY: build clean test test-coverage install lint fmt vet ci-local

BINARY  := bin/shelfctl
MAIN    := ./cmd/shelfctl

build:
	@mkdir -p bin
	go build -o $(BINARY) $(MAIN)

test:
	go test -v ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

install:
	go install $(MAIN)

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
