BINARY  := planwerk-review
MAIN    := ./cmd/planwerk-review
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test vet lint fmt clean completions man

all: lint test build

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

completions:
	mkdir -p completions
	go run $(MAIN) completion bash > completions/$(BINARY).bash
	go run $(MAIN) completion zsh > completions/_$(BINARY)
	go run $(MAIN) completion fish > completions/$(BINARY).fish

man:
	mkdir -p docs/man
	go run $(MAIN) gen-man-pages docs/man

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
	rm -rf completions docs/man
