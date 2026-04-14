# syntax=docker/dockerfile:1
# GoReleaser runtime image — binary is pre-built and provided by GoReleaser
FROM alpine:3.21

RUN apk add --no-cache wireguard-tools iptables ca-certificates && \
    mkdir -p /etc/wgrift /var/lib/wgrift /etc/wireguard

COPY wgrift /usr/local/bin/wgrift

COPY <<'ENTRYPOINT' /usr/local/bin/docker-entrypoint.sh
#!/bin/sh
set -e

# Create directories if missing
mkdir -p /etc/wgrift /var/lib/wgrift /etc/wireguard

# Copy default config on first run
if [ ! -f /etc/wgrift/config.yaml ]; then
    cp /etc/wgrift/config.yaml.default /etc/wgrift/config.yaml
    echo "==> Installed default config at /etc/wgrift/config.yaml"
fi

# Require master key — either via env var or file
if [ -z "${WGRIFT_MASTER_KEY:-}" ] && [ ! -f /etc/wgrift/master.key ]; then
    echo "ERROR: No master encryption key provided." >&2
    echo "" >&2
    echo "The master key encrypts peer private keys and secrets at rest." >&2
    echo "Provide one of the following:" >&2
    echo "" >&2
    echo "  1. Set the WGRIFT_MASTER_KEY environment variable:" >&2
    echo "     docker run -e WGRIFT_MASTER_KEY=\$(head -c 32 /dev/urandom | base64) ..." >&2
    echo "" >&2
    echo "  2. Mount a key file to /etc/wgrift/master.key:" >&2
    echo "     head -c 32 /dev/urandom | base64 > master.key" >&2
    echo "     docker run -v \$(pwd)/master.key:/etc/wgrift/master.key:ro ..." >&2
    echo "" >&2
    exit 1
fi

exec wgrift "$@"
ENTRYPOINT

COPY <<'CONFIG' /etc/wgrift/config.yaml.default
server:
  listen: "0.0.0.0:8080"
  external_url: ""
  tls:
    mode: none

database:
  path: /var/lib/wgrift/wgrift.db

encryption:
  master_key_file: /etc/wgrift/master.key

auth:
  session_timeout: 30m
  max_session_age: 24h
  local:
    enabled: true
    min_password_length: 16

logging:
  connection_poll_interval: 30s
  connection_timeout: 180s
  retention_days: 90
CONFIG

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

VOLUME ["/etc/wgrift", "/var/lib/wgrift", "/etc/wireguard"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["serve", "--config", "/etc/wgrift/config.yaml"]
