#!/bin/bash
set -euo pipefail

# wgRift installer for Proxmox LXC (Debian/Ubuntu)
# Run inside the LXC container as root

echo "==> Installing wgRift"

# Install wireguard-tools
apt-get update -qq
apt-get install -y -qq wireguard-tools

# Create directories
mkdir -p /opt/wgrift/web
mkdir -p /etc/wgrift
mkdir -p /var/lib/wgrift

# Copy files
cp wgrift /opt/wgrift/wgrift
chmod +x /opt/wgrift/wgrift

cp wgrift.wasm /opt/wgrift/web/
cp index.html /opt/wgrift/web/
cp wasm_exec.js /opt/wgrift/web/

# Generate master key if not exists
if [ ! -f /etc/wgrift/master.key ]; then
    head -c 32 /dev/urandom | base64 > /etc/wgrift/master.key
    chmod 600 /etc/wgrift/master.key
    echo "==> Generated master key at /etc/wgrift/master.key"
fi

# Install config if not exists
if [ ! -f /etc/wgrift/config.yaml ]; then
    cp config.yaml /etc/wgrift/config.yaml
    echo "==> Installed config at /etc/wgrift/config.yaml"
    echo "    Edit external_url in /etc/wgrift/config.yaml before starting"
fi

# Install systemd service
cp wgrift.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable wgrift

echo ""
echo "==> wgRift installed!"
echo ""
echo "Next steps:"
echo "  1. Edit /etc/wgrift/config.yaml (set external_url)"
echo "  2. systemctl start wgrift"
echo "  3. Open http://<lxc-ip>:8443 to set up admin account"
echo ""
echo "Proxmox host requirement (run on the host, not in LXC):"
echo "  modprobe wireguard && echo wireguard >> /etc/modules"
