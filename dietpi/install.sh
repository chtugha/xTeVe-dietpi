#!/bin/bash
set -euo pipefail

trap 'rm -f /tmp/xteve' EXIT

BINARY_DEST='/usr/local/bin/xteve'
DATA_DIR='/mnt/dietpi_userdata/xteve'
SERVICE_DEST='/etc/systemd/system/xteve.service'

ARCH=$(dpkg --print-architecture)
case "$ARCH" in
    armhf)       ARCH=arm ;;
    amd64|arm64) ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

DOWNLOAD_URL="https://github.com/whisper/xTeVe-dietpi/releases/latest/download/xteve_linux_${ARCH}"

echo "Downloading xTeVe binary for linux/${ARCH}..."
curl -fsSL "$DOWNLOAD_URL" -o /tmp/xteve
install -m 0755 /tmp/xteve "$BINARY_DEST"

EXTRA_GROUPS=''
getent group render &>/dev/null && EXTRA_GROUPS=',render' || true
id -u xteve &>/dev/null || \
    useradd --system \
            --home-dir "$DATA_DIR" \
            --shell /usr/sbin/nologin \
            --groups "video${EXTRA_GROUPS}" \
            xteve

mkdir -p "$DATA_DIR"
chown xteve:xteve "$DATA_DIR"
chmod 0750 "$DATA_DIR"

cat > "$SERVICE_DEST" << 'EOF'
[Unit]
Description=xTeVe M3U Proxy for Plex DVR and Emby Live TV
Documentation=https://github.com/whisper/xTeVe-dietpi
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=xteve
Group=xteve
Environment="DIETPI=1"
WorkingDirectory=/mnt/dietpi_userdata/xteve
ExecStart=/usr/local/bin/xteve --config /mnt/dietpi_userdata/xteve/
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
chmod 0644 "$SERVICE_DEST"

systemctl daemon-reload
systemctl enable --now xteve

# G_WHIP_YESNO is a DietPi shell function available only when this script is
# sourced inside dietpi-software's own environment. When invoked as a
# subprocess (the typical case), declare -f will return non-zero and this
# block is skipped. To install FFmpeg manually, run:
#   dietpi-software install 7
if declare -f G_WHIP_YESNO > /dev/null 2>&1; then
    if G_WHIP_YESNO 'Would you like to install FFmpeg for stream buffering/re-encoding?\n\nThis is only required when "Buffer" is set to "FFmpeg" in xTeVe settings.'; then
        G_DIETPI-INSTALL_SOFTWARE 7
    fi
fi
