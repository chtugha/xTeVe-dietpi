# xTeVe — DietPi Integration Guide

xTeVe is an M3U proxy that presents your IPTV playlists as a network HDHomeRun tuner to Plex DVR and Emby Live TV. This guide covers everything needed to install, configure, and maintain xTeVe on a DietPi system.

---

## Prerequisites

| Requirement | Notes |
|---|---|
| DietPi (Bullseye or Bookworm) | Tested on `amd64`, `arm64`, and `armhf` (ARMv7) |
| systemd | Available on all DietPi builds |
| `curl` | Pre-installed on DietPi |
| Plex Media Server ≥ 1.11.1.4730 **or** Emby Server ≥ 3.5.3.0 | At least one required |
| Plex Pass **or** Emby Premiere | Required for DVR / Live TV features |

Optional:

- **FFmpeg** (DietPi software ID 7) — required only when the xTeVe *Buffer* setting is set to `FFmpeg`
- **VLC** (`apt install vlc`) — required only when *Buffer* is set to `VLC`

---

## Installation

### Via dietpi-software

```bash
dietpi-software install <ID>
```

> Replace `<ID>` with the xTeVe software ID assigned in your DietPi release. The installer script (`dietpi/install.sh`) handles all steps below automatically.

### Manual installation (advanced)

If you are installing outside of `dietpi-software`, run the install script directly as root:

```bash
sudo bash dietpi/install.sh
```

The script performs the following steps:

1. Detects the CPU architecture (`amd64`, `arm64`, or `armhf`/`arm`).
2. Downloads the pre-built binary for that architecture from [GitHub Releases](https://github.com/whisper/xTeVe-dietpi/releases/latest).
3. Installs the binary to `/usr/local/bin/xteve` (mode `0755`).
4. Creates a system user `xteve` (no login shell, home at `/mnt/dietpi_userdata/xteve`).
5. Creates the data directory `/mnt/dietpi_userdata/xteve/` (mode `0750`, owned by `xteve:xteve`).
6. Installs the systemd unit `systemd/xteve.service` to `/etc/systemd/system/xteve.service`.
7. Enables and starts the service.
8. Optionally installs FFmpeg if you answer **Yes** to the interactive prompt.

---

## Service Management

All standard `systemctl` commands apply:

```bash
# Check service status and recent log output
systemctl status xteve

# Start / stop / restart
systemctl start  xteve
systemctl stop   xteve
systemctl restart xteve

# Enable / disable auto-start on boot
systemctl enable  xteve
systemctl disable xteve
```

View logs from the current boot:

```bash
journalctl -u xteve -b
```

Follow logs in real time:

```bash
journalctl -u xteve -f
```

---

## Data Directory Layout

All xTeVe runtime data lives under `/mnt/dietpi_userdata/xteve/` and is **preserved across reinstalls and upgrades**.

```
/mnt/dietpi_userdata/xteve/
├── backup/                  # Automatic and manual backups (.zip archives)
├── cache/
│   └── images/              # Cached channel logo images
├── data/
│   ├── images/              # User-uploaded channel logos
│   ├── xteve.m3u            # Generated M3U playlist
│   └── xteve.xml            # Generated XMLTV EPG file
├── authentication.json      # User accounts and API tokens
├── pms.json                 # Plex / Emby device mapping
├── settings.json            # All user-configurable settings
├── urls.json                # Provider URL cache
└── xepg.json                # XEPG channel database
```

---

## Default Port and How to Change It

xTeVe listens on **port 34400** by default. The web UI is reachable at:

```
http://<dietpi-ip>:34400/web/
```

To use a different port:

1. Open the web UI → **Settings** → **Port** and change the value, then save.
2. Alternatively, pass the port on the command line by editing the `ExecStart` line in `/etc/systemd/system/xteve.service`:
   ```ini
   ExecStart=/usr/local/bin/xteve --config /mnt/dietpi_userdata/xteve/ --port 8080
   ```
3. Reload and restart:
   ```bash
   systemctl daemon-reload && systemctl restart xteve
   ```

> **Firewall note**: If your DietPi uses `ufw` or `iptables`, open the chosen port for inbound TCP connections from your Plex/Emby host.

---

## Optional Dependencies

### FFmpeg (stream buffering and re-encoding)

FFmpeg is required only when the xTeVe **Buffer** setting (in Settings → Streaming) is set to `FFmpeg`. Install via DietPi:

```bash
dietpi-software install 7
```

The xTeVe installer will also prompt you to install FFmpeg at the end of installation.

### VLC (alternative buffer backend)

```bash
apt install vlc
```

---

## Uninstall

### Via dietpi-software

```bash
dietpi-software uninstall <ID>
```

### Manual uninstall

```bash
sudo bash dietpi/uninstall.sh
```

This will:
1. Stop and disable the `xteve` service.
2. Remove `/etc/systemd/system/xteve.service`.
3. Remove `/usr/local/bin/xteve`.
4. Remove the `xteve` system user.

> **Data preservation**: The directory `/mnt/dietpi_userdata/xteve/` and all its contents (settings, channel database, backups) are **intentionally kept** after uninstall. Remove it manually if you want a clean slate:
> ```bash
> rm -rf /mnt/dietpi_userdata/xteve/
> ```

---

## Upgrade

### Tag-driven CI release (recommended)

Every push of a version tag (e.g., `v2.2.1`) to the repository triggers a GitHub Actions workflow that cross-compiles binaries for all three architectures and publishes them as GitHub Release assets. The DietPi install script always downloads from `releases/latest`, so reinstalling fetches the newest build:

```bash
dietpi-software reinstall <ID>
# or manually:
sudo bash dietpi/install.sh
```

### In-application auto-update

xTeVe has a built-in self-updater. **On DietPi, this is disabled by default** (`XteveAutoUpdate = false`) to prevent the binary from being replaced outside of DietPi package management.

If you want to enable the in-application updater (not recommended on DietPi), go to **Settings → xTeVe Auto Update** and enable it. A warning will be logged each time an update runs, reminding you that the binary was replaced outside of `dietpi-software`.

---

## Troubleshooting

### xTeVe fails to start

Check the journal for errors:

```bash
journalctl -u xteve -b --no-pager
```

Common causes:

- **Port already in use** — another process is bound to port 34400. Change the port (see above) or stop the conflicting process.
- **Permission denied on data directory** — ensure `/mnt/dietpi_userdata/xteve/` is owned by `xteve:xteve`:
  ```bash
  chown -R xteve:xteve /mnt/dietpi_userdata/xteve/
  chmod 0750 /mnt/dietpi_userdata/xteve/
  ```

### Web UI is not reachable

1. Confirm the service is running: `systemctl status xteve`
2. Confirm xTeVe is listening: `ss -tlnp | grep 34400`
3. Check that your firewall allows inbound TCP on port 34400 from the Plex/Emby host.

### Plex or Emby cannot find the tuner

- Plex discovers HDHomeRun tuners via SSDP (UDP). Ensure multicast traffic is not blocked between xTeVe and Plex.
- The xTeVe web UI shows the HDHomeRun discovery URL under **Settings → HDHomeRun**. Add it manually in Plex under **Settings → Live TV & DVR → Set Up Plex Tuner → Enter device address**.

### Log access

The xTeVe web UI includes a built-in log viewer at **Log** in the top navigation bar. Logs are also available via `journalctl -u xteve`.
