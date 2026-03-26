# xTeVe — DietPi-Software install block
#
# This file is a REFERENCE IMPLEMENTATION showing the code that would be added
# inside Install_Software() in dietpi/dietpi-software for an upstream PR to
# MichaIng/DietPi. It is NOT a standalone script — it is sourced by
# dietpi-software and relies on its helper functions and variables.
#
# Software ID: TBD (assigned by DietPi maintainers)
# Category: 2 (Media)
# Dependencies: 7 (FFmpeg, optional — listed in aSOFTWARE_DEPS)

# --- Software_Arrays_Init() registration block ---
# software_id=<TBD>
# aSOFTWARE_NAME[$software_id]='xTeVe'
# aSOFTWARE_DESC[$software_id]='M3U proxy for Plex DVR and Emby/Jellyfin Live TV'
# aSOFTWARE_CATX[$software_id]=2
# aSOFTWARE_DOCS[$software_id]='https://github.com/chtugha/xTeVe-dietpi'

# --- Install_Software() block ---
if To_Install $software_id xteve # xTeVe
then
	# Architecture mapping
	case $G_HW_ARCH in
		2) local arch='arm';;
		3) local arch='arm64';;
		*) local arch='amd64';;
	esac

	# Download binary
	local fallback_url="https://github.com/chtugha/xTeVe-dietpi/releases/download/v2.2.0/xteve_linux_$arch"
	Download_Install "$(curl -sSfL 'https://api.github.com/repos/chtugha/xTeVe-dietpi/releases/latest' | grep -Po "\"browser_download_url\": *\"\K[^\"]*\/xteve_linux_$arch(?=\")")" /usr/local/bin/xteve

	G_EXEC chmod +x /usr/local/bin/xteve

	# Data directory
	G_EXEC mkdir -p /mnt/dietpi_userdata/xteve

	# User
	Create_User -G video -d /mnt/dietpi_userdata/xteve xteve

	# Permissions
	G_EXEC chown -R xteve:xteve /mnt/dietpi_userdata/xteve

	# Service
	cat << '_EOF_' > /etc/systemd/system/xteve.service
[Unit]
Description=xTeVe M3U Proxy for Plex DVR and Emby Live TV (DietPi)
Documentation=https://github.com/chtugha/xTeVe-dietpi
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
ProtectSystem=full
ReadWritePaths=/mnt/dietpi_userdata/xteve

[Install]
WantedBy=multi-user.target
_EOF_
fi
