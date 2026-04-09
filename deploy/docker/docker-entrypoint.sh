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
