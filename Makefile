BINARY := ghw
MODULE := github.com/anomalyco/gitwhy
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build test clean lint vet

all: build

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) $(MODULE)

test:
	go test ./... -count=1 -v

test-short:
	go test ./... -count=1

vet:
	go vet ./...

lint:
	golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed"

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

run:
	go run . $(ARGS)
