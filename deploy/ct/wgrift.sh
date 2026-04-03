#!/usr/bin/env bash

# wgRift LXC installer for Proxmox VE
# Usage: bash -c "$(curl -fsSL https://raw.githubusercontent.com/drudge/wgrift/main/deploy/ct/wgrift.sh)"

set -euo pipefail

# --- Colors ---
RD='\033[0;31m'
GN='\033[0;32m'
YW='\033[0;33m'
BL='\033[0;34m'
CL='\033[0m'

# --- Config ---
APP="wgRift"
REPO="drudge/wgrift"
var_os="debian"
var_version="12"
var_cpu="${WGRIFT_CPU:-1}"
var_ram="${WGRIFT_RAM:-256}"
var_disk="${WGRIFT_DISK:-2}"
var_hostname="${WGRIFT_HOSTNAME:-wgrift}"
var_storage="${WGRIFT_STORAGE:-local-lvm}"
var_bridge="${WGRIFT_BRIDGE:-vmbr0}"

# --- Functions ---
msg_info() { echo -e " ${BL}[INFO]${CL} $1"; }
msg_ok() { echo -e " ${GN}[OK]${CL} $1"; }
msg_warn() { echo -e " ${YW}[WARN]${CL} $1"; }
msg_error() { echo -e " ${RD}[ERROR]${CL} $1"; exit 1; }

header() {
  cat <<'EOF'
               ____  _ ______
 _      _____ / __ \(_) __/ /_
| | /| / / _ \/ /_/ / / /_/ __/
| |/ |/ /  __/ _, _/ / __/ /_
|__/|__/\___/_/ |_/_/_/  \__/

  WireGuard Management Platform
EOF
  echo ""
}

check_root() {
  if [[ $EUID -ne 0 ]]; then
    msg_error "This script must be run as root on the Proxmox host"
  fi
}

check_pve() {
  if ! command -v pct &>/dev/null; then
    msg_error "This script must be run on a Proxmox VE host"
  fi
}

get_latest_release() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

get_next_ctid() {
  pvesh get /cluster/nextid 2>/dev/null || echo "100"
}

get_template() {
  local template
  template=$(pveam list local 2>/dev/null | grep "debian-12-standard" | sort -V | tail -1 | awk '{print $1}')
  if [[ -z "$template" ]]; then
    msg_info "Downloading Debian 12 template..."
    pveam update &>/dev/null
    pveam download local debian-12-standard_12.7-1_amd64.tar.zst &>/dev/null || \
      pveam download local "$(pveam available | grep 'debian-12-standard' | awk '{print $2}' | tail -1)" &>/dev/null
    template=$(pveam list local | grep "debian-12-standard" | sort -V | tail -1 | awk '{print $1}')
  fi
  echo "$template"
}

# --- Main ---
header
check_root
check_pve

# Check wireguard kernel module
if ! lsmod | grep -q wireguard; then
  msg_info "Loading wireguard kernel module..."
  modprobe wireguard || msg_error "Failed to load wireguard module. Install it: apt install wireguard-dkms"
  if ! grep -q wireguard /etc/modules 2>/dev/null; then
    echo wireguard >> /etc/modules
  fi
  msg_ok "Wireguard module loaded"
fi

# Get release version
msg_info "Checking latest release..."
VERSION=$(get_latest_release)
if [[ -z "$VERSION" ]]; then
  msg_error "Could not determine latest release. Check https://github.com/${REPO}/releases"
fi
msg_ok "Latest version: ${VERSION}"

# Detect architecture
ARCH=$(dpkg --print-architecture)
case $ARCH in
  amd64) ARCH_SUFFIX="linux-amd64" ;;
  arm64) ARCH_SUFFIX="linux-arm64" ;;
  *) msg_error "Unsupported architecture: $ARCH" ;;
esac

# Prompt for settings
echo ""
echo -e "${BL}Container Settings${CL} (press Enter for defaults)"
echo ""
read -rp "  Hostname [${var_hostname}]: " input_hostname
var_hostname="${input_hostname:-$var_hostname}"

read -rp "  CPU cores [${var_cpu}]: " input_cpu
var_cpu="${input_cpu:-$var_cpu}"

read -rp "  RAM in MB [${var_ram}]: " input_ram
var_ram="${input_ram:-$var_ram}"

read -rp "  Disk in GB [${var_disk}]: " input_disk
var_disk="${input_disk:-$var_disk}"

read -rp "  Storage [${var_storage}]: " input_storage
var_storage="${input_storage:-$var_storage}"

read -rp "  Bridge [${var_bridge}]: " input_bridge
var_bridge="${input_bridge:-$var_bridge}"

CTID=$(get_next_ctid)
read -rp "  Container ID [${CTID}]: " input_ctid
CTID="${input_ctid:-$CTID}"

echo ""

# Get template
msg_info "Preparing Debian 12 template..."
TEMPLATE=$(get_template)
if [[ -z "$TEMPLATE" ]]; then
  msg_error "Could not find Debian 12 template"
fi
msg_ok "Template: ${TEMPLATE}"

# Create container
msg_info "Creating LXC container ${CTID} (${var_hostname})..."
pct create "$CTID" "$TEMPLATE" \
  --hostname "$var_hostname" \
  --cores "$var_cpu" \
  --memory "$var_ram" \
  --rootfs "${var_storage}:${var_disk}" \
  --net0 "name=eth0,bridge=${var_bridge},ip=dhcp" \
  --ostype "$var_os" \
  --unprivileged 0 \
  --features nesting=1 \
  --onboot 1 \
  --start 0 \
  &>/dev/null
msg_ok "Container created"

# Start container
msg_info "Starting container..."
pct start "$CTID"
sleep 3
msg_ok "Container started"

# Install inside container
msg_info "Installing wgRift ${VERSION}..."

pct exec "$CTID" -- bash -c "
  set -e
  export DEBIAN_FRONTEND=noninteractive

  # Update and install deps
  apt-get update -qq
  apt-get install -y -qq wireguard-tools curl

  # Create directories
  mkdir -p /etc/wgrift /var/lib/wgrift

  # Download release
  cd /tmp
  curl -fsSL 'https://github.com/${REPO}/releases/download/${VERSION}/wgrift-${VERSION}-${ARCH_SUFFIX}.tar.gz' -o wgrift.tar.gz
  tar xzf wgrift.tar.gz
  cd wgrift-${VERSION}-${ARCH_SUFFIX}

  # Install binary
  cp wgrift /usr/local/bin/wgrift
  chmod +x /usr/local/bin/wgrift

  # Generate master key
  head -c 32 /dev/urandom | base64 > /etc/wgrift/master.key
  chmod 600 /etc/wgrift/master.key

  # Install config
  cp config.yaml /etc/wgrift/config.yaml

  # Install systemd service
  cp wgrift.service /etc/systemd/system/
  systemctl daemon-reload
  systemctl enable wgrift

  # Start
  systemctl start wgrift

  # Cleanup
  rm -rf /tmp/wgrift*
"
msg_ok "wgRift installed"

# Get container IP
sleep 2
CT_IP=$(pct exec "$CTID" -- hostname -I 2>/dev/null | awk '{print $1}')

echo ""
echo -e "${GN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${CL}"
echo -e "${GN} wgRift installed successfully!${CL}"
echo -e "${GN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${CL}"
echo ""
echo -e "  Container ID:  ${BL}${CTID}${CL}"
echo -e "  Hostname:      ${BL}${var_hostname}${CL}"
if [[ -n "$CT_IP" ]]; then
  echo -e "  Web UI:        ${BL}http://${CT_IP}:8080${CL}"
fi
echo ""
echo -e "  ${YW}Next steps:${CL}"
echo -e "  1. Open the web UI and create your admin account"
echo -e "  2. Set your public endpoint in interface settings"
echo -e "  3. Import existing configs or create new interfaces"
echo ""
echo -e "  Config:  /etc/wgrift/config.yaml"
echo -e "  Logs:    journalctl -u wgrift -f"
echo ""
