.PHONY: build clean test install

BINARY  := bin/shelfctl
MAIN    := ./cmd/shelfctl

build:
	go build -o $(BINARY) $(MAIN)

test:
	go test ./...

install:
	go install $(MAIN)

clean:
	rm -rf bin/

.DEFAULT_GOAL := build
