BINARY  := planwerk-review
MAIN    := ./cmd/planwerk-review
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test vet lint fmt clean

all: lint test build

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -s -w .

clean:
	rm -f $(BINARY)
