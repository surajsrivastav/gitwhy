BINARY := ghw
MODULE := github.com/surajsrivastav/gitwhy
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build install test test-short vet lint coverage clean run release snapshot

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

release:
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
	git push origin refs/notes/gitwhy

snapshot:
	goreleaser release --snapshot --clean 2>/dev/null || echo "goreleaser not installed (brew install goreleaser)"
