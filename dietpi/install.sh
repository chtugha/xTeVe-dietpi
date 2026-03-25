#!/bin/bash
set -euo pipefail

trap 'rm -f /tmp/xteve' EXIT

# BASH_SOURCE[0] resolves correctly when invoked as a subprocess (e.g. via
# G_EXEC bash dietpi/install.sh, which is the DietPi dietpi-software model).
# Direct interactive sourcing from a different directory is not supported.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_DEST='/usr/local/bin/xteve'
DATA_DIR='/mnt/dietpi_userdata/xteve'
SERVICE_SRC="$SCRIPT_DIR/../systemd/xteve.service"
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

install -m 0644 "$SERVICE_SRC" "$SERVICE_DEST"

systemctl daemon-reload
systemctl enable --now xteve

if G_WHIP_YESNO 'Would you like to install FFmpeg for stream buffering/re-encoding?\n\nThis is only required when "Buffer" is set to "FFmpeg" in xTeVe settings.'; then
    G_DIETPI-INSTALL_SOFTWARE 7
fi
