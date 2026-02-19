BINARY := agrev
MODULE := github.com/sprite-ai/agrev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-s -w \
	-X $(MODULE)/internal/cli.version=$(VERSION) \
	-X $(MODULE)/internal/cli.commit=$(COMMIT) \
	-X $(MODULE)/internal/cli.date=$(DATE)"

.PHONY: build test lint install clean fmt vet

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/agrev

install:
	go install $(LDFLAGS) ./cmd/agrev

test:
	go test -race ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

clean:
	rm -rf bin/ dist/
