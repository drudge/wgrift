VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
           -X github.com/drudge/wgrift/pkg/version.Version=$(VERSION) \
           -X github.com/drudge/wgrift/pkg/version.Commit=$(COMMIT) \
           -X github.com/drudge/wgrift/pkg/version.Date=$(DATE)

.PHONY: build test lint clean wasm serve-web

build:
	go build -ldflags "$(LDFLAGS)" -o bin/wgrift ./cmd/wgrift

test:
	go test ./internal/...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

wasm:
	GOOS=js GOARCH=wasm go build -ldflags "$(LDFLAGS)" -o ui/web/wgrift.wasm ./ui/web

serve-web: wasm
	go run ./cmd/serve-web
