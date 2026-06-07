<div align="center" style="background-color: #111; padding: 100px;">
    <img width="880" height="200" src="html/img/logo_b_880x200.jpg" alt="xTeVe" />
</div>
<br>

# xTeVe for DietPi & Standard Systems

An enhanced, high-performance fork of **xTeVe** — the ultimate M3U Proxy for Plex DVR, Emby, and Jellyfin Live TV. This fork is optimized for both standard operating systems and **DietPi** environments, featuring critical fixes for streaming and updating.

## Credits & Original Creator

This project is a continuation and enhancement of the original **xTeVe** software. We give full credit and appreciation to the original creator, **xteve-project**, and the incredible work they did in building this tool.

- **Original Repository**: [https://github.com/xteve-project/xTeVe](https://github.com/xteve-project/xTeVe)
- **Original Documentation**: [https://github.com/xteve-project/xTeVe-Documentation](https://github.com/xteve-project/xTeVe-Documentation)

---

## Key Fork Enhancements

- **Plex Restreaming & Buffering Fix**: Fully resolved the critical issue where restreaming/buffering mode failed to stream to Plex DVR or strict HTTP clients like FFmpeg and VLC. The server no longer forces empty `Content-Length` headers, allowing unconstrained, real-time chunked stream consumption.
- **GitHub Release Updater**: Completely rebuilt the self-updater to directly query the GitHub Releases API. It detects your current OS/architecture, fetches the correct binary automatically, and hot-swaps it.
- **Manual Update Check Trigger**: Added an **Update xTeVe** button in the Web UI Settings next to the auto-update toggle, letting you check for and execute updates manually on demand.
- **DietPi Safety Checks**: Built-in safety features prevent accidental package manager conflicts on DietPi installations. Auto-updates default to disabled when `DIETPI=1` is detected, with log warning level checks to notify users of out-of-band updates.

---

## Features

- **Virtual Tuner Emulation**: Emulates a SiliconDust HDHomeRun network tuner to integrate seamlessly with Plex Media Server, Emby, and Jellyfin.
- **Stream Merging**: Merge multiple external M3U playlists and XMLTV EPG sources into a single coherent lineup.
- **Stream Buffering**: Built-in HLS/M3U8 buffering engines and FFmpeg wrapper support to prevent connection drops.
- **Lineup Customization**: Manage channel numbers, names, ordering, category tags, logos, and EPG mapping through an elegant Web UI.

---

## Installation Guide

### Method 1: Installing Pre-compiled Releases (Recommended)

Standalone, statically compiled binaries are available for multiple operating systems and architectures.

1. Navigate to our [Releases Page](https://github.com/chtugha/xTeVe-dietpi/releases).
2. Download the binary that matches your operating system and architecture:
   - **Linux (64-bit Intel/AMD)**: `xteve_linux_amd64` or `xteve_linux_amd64.tar.gz`
   - **Linux (64-bit ARM / Raspberry Pi 4/5)**: `xteve_linux_arm64` or `xteve_linux_arm64.tar.gz`
   - **Linux (32-bit ARM / Raspberry Pi 3)**: `xteve_linux_arm` or `xteve_linux_arm.tar.gz`
3. Extract and make the binary executable:
   ```bash
   chmod +x xteve_linux_amd64
   ./xteve_linux_amd64
   ```

---

### Method 2: DietPi Service Setup (DietPi Systems)

If you are running on **DietPi**, you can integrate xTeVe as a systemd background service. We provide a reference installer script in `./dietpi/install.sh` that is fully compliant with the `dietpi-software` manager environment.

To set up the systemd service manually on your DietPi machine:

1. Create a dedicated application directory:
   ```bash
   mkdir -p /mnt/dietpi_userdata/xteve
   ```
2. Download the binary and set execute permissions:
   ```bash
   curl -sSfL 'https://api.github.com/repos/chtugha/xTeVe-dietpi/releases/latest' \
     | grep -Po '"browser_download_url": *"\K[^"]*\/xteve_linux_amd64' \
     | xargs curl -L -o /usr/local/bin/xteve
   chmod +x /usr/local/bin/xteve
   ```
3. Create a dedicated non-privileged user:
   ```bash
   useradd -r -s /usr/sbin/nologin -G video -d /mnt/dietpi_userdata/xteve xteve
   chown -R xteve:xteve /mnt/dietpi_userdata/xteve
   ```
4. Install and enable the systemd service template found in `./dietpi/install.sh`:
   ```bash
   # Enable the systemd service
   systemctl daemon-reload
   systemctl enable --now xteve
   ```

To uninstall or remove the service, refer to `./dietpi/uninstall.sh` to cleanly stop, disable, and clean up the directories.

---

### Method 3: Compiling from Source

To compile the latest version of xTeVe directly from the source code, make sure you have Go installed (Go version 1.21 or newer is required).

1. Clone the repository:
   ```bash
   git clone https://github.com/chtugha/xTeVe-dietpi.git
   cd xTeVe-dietpi
   ```
2. Build the project using the `./Makefile`:
   - Build for your local architecture:
     ```bash
     make build
     ```
   - Compile for all supported platforms (Linux AMD64, ARM64, and ARMv7):
     ```bash
     make build-all
     ```
3. The compiled binaries will be outputted to the `build/` directory.

---

## Command Line Arguments

When launching the binary, you can customize the configuration and server properties using command-line flags:

```text
  -config <path>    : Path to the configuration folder (default: ~/.xteve/)
  -port <number>    : HTTP server port (default: 34400)
  -branch <name>    : Update branch [master|beta] (default: master)
  -debug <level>    : Set debug level [0 - 3] (default: 0)
  -info             : Output system architecture and version info, then exit
  -restore <path>   : Path to a backup ZIP archive to restore from on startup
  -h                : Show help menu and list all options
```

---

## Plex DVR & Emby Integration

1. Launch xTeVe on your server.
2. Open your web browser and go to `http://<your-server-ip>:34400/web/`.
3. Follow the Setup Wizard to point to your `.m3u` playlist and `.xml` EPG files.
4. Once configured, open **Plex Media Server** and navigate to **Settings** > **Live TV & DVR**.
5. Click **Add Device** / **Set Up DVR**. Plex should automatically discover the xTeVe virtual tuner. If not, enter the address manually as `http://<your-server-ip>:34400/`.
6. Select your channel lineup and associate it with the XMLTV guide provided by xTeVe at `http://<your-server-ip>:34400/xmltv/xteve.xml`.

---

## Contributing & Development

For details on project structure, internal packages, and guidelines for testing, see `./README-DEV.md`.

*Note: For security and stability, the automatic self-update feature can be toggled in the Web UI Settings panel. Under DietPi environments, updates can be performed safely via the manual trigger button.*
