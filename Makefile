VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION) -X 'github.com/sreeram/gurl/internal/cli/commands.CurrentVersion=$(VERSION)'
BINARY := gurl
CMD := ./cmd/gurl

.PHONY: build install clean test test-race

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

install: build
	cp $(BINARY) $(shell which gurl 2>/dev/null || echo "$(shell go env GOPATH)/bin/$(BINARY)")

clean:
	rm -f $(BINARY)

test:
	go test -count=1 -timeout 5m ./...

test-race:
	go test -race -count=1 -timeout 10m ./...
