VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
           -X github.com/drudge/wgrift/pkg/version.Version=$(VERSION) \
           -X github.com/drudge/wgrift/pkg/version.Commit=$(COMMIT) \
           -X github.com/drudge/wgrift/pkg/version.Date=$(DATE)

.PHONY: build test lint clean wasm serve-web serve dist

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

serve: wasm build
	WGRIFT_MASTER_KEY=$${WGRIFT_MASTER_KEY:-dev-master-key} ./bin/wgrift serve

dist: wasm
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/wgrift ./cmd/wgrift
	cp deploy/wgrift.service dist/
	cp deploy/config.yaml dist/
	cp deploy/install.sh dist/
	@echo "Distribution files ready in dist/"
	@echo "  Binary has embedded web assets — single file deploy"
