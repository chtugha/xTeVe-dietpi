#!/bin/bash
set -euo pipefail

systemctl disable --now xteve || true

rm -f /etc/systemd/system/xteve.service

systemctl daemon-reload

rm -f /usr/local/bin/xteve

userdel xteve || true

echo "Note: xTeVe user data in /mnt/dietpi_userdata/xteve/ has been preserved."
