# Build stage
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache brotli git

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Build WASM UI assets
RUN GOOS=js GOARCH=wasm GOWASM=satconv,signext go build \
    -ldflags "-s -w \
      -X github.com/drudge/wgrift/pkg/version.Version=${VERSION} \
      -X github.com/drudge/wgrift/pkg/version.Commit=${COMMIT} \
      -X github.com/drudge/wgrift/pkg/version.Date=${DATE}" \
    -o ui/web/wgrift.wasm ./ui/web

# Copy wasm_exec.js (lib/wasm in Go 1.24+, misc/wasm in older)
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ui/web/wasm_exec.js 2>/dev/null || \
    cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" ui/web/wasm_exec.js

# Compress WASM
RUN gzip -9 -k -f ui/web/wgrift.wasm && \
    brotli -9 -k -f ui/web/wgrift.wasm

# Build server binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w \
      -X github.com/drudge/wgrift/pkg/version.Version=${VERSION} \
      -X github.com/drudge/wgrift/pkg/version.Commit=${COMMIT} \
      -X github.com/drudge/wgrift/pkg/version.Date=${DATE}" \
    -o /out/wgrift ./cmd/wgrift

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache wireguard-tools ca-certificates && \
    mkdir -p /etc/wgrift /var/lib/wgrift /etc/wireguard

COPY --from=builder /out/wgrift /usr/local/bin/wgrift
COPY deploy/docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY deploy/config.yaml /etc/wgrift/config.yaml.default

VOLUME ["/etc/wgrift", "/var/lib/wgrift", "/etc/wireguard"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["serve", "--config", "/etc/wgrift/config.yaml"]
