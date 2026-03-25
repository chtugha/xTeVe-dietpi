# Product Requirements Document: xTeVe DietPi Software Package

## 1. Background

**xTeVe** is an M3U proxy server for Plex DVR and Emby Live TV, written in Go. This repository is a fork of the upstream `xteve-project/xTeVe` with incremental improvements (dead-code removal, bug fixes, buffer improvements, SSDP goroutine leak fix). The goal of this task is to make this repository the canonical source for a **first-class DietPi software package** — meaning it can be installed, configured, and managed through DietPi's `dietpi-software` tool and integrates cleanly with DietPi's conventions, lifecycle management, and user expectations.

---

## 2. Problem Statement

The current codebase is a standalone Go application with no DietPi-specific integration. The following gaps prevent it from being used as a DietPi software package:

1. **No systemd service file** — xTeVe cannot be started, stopped, or auto-started as a system service.
2. **Self-updating binary** — xTeVe downloads and hot-swaps its own binary from GitHub via `syscall.Exec`, which breaks systemd process tracking and conflicts with DietPi's package management lifecycle.
3. **Wrong data directory** — Default config is `~/.xteve/`, but DietPi convention requires user data under `/mnt/dietpi_userdata/xteve/`.
4. **No dedicated system user** — DietPi installs services under dedicated unprivileged users, not the login user.
5. **No install/uninstall scripts** — No DietPi-software Bash integration exists.
6. **No build pipeline** — No Makefile, no CI workflow for multi-arch binary builds (ARM/ARM64/amd64).
7. **Several code-quality issues** — Deprecated API usage, magic numbers, dead code, and a `go.mod` toolchain declaration that specifies Go 1.25+/1.26+ (valid releases, but not available via Debian apt on DietPi) require attention before the package is production-ready.

---

## 3. Goals

- Provide a complete, first-class DietPi software package for xTeVe.
- Fix all identified bugs, deprecated API usage, magic numbers, stubs, and dead code in the Go source.
- Add thorough in-code and external documentation for ongoing development and end-user operation.
- Ensure future compatibility with DietPi's distribution model (install, update, uninstall, automated install, survey).

---

## 4. Non-Goals

- Porting the web UI from JavaScript to any other framework.
- Adding new xTeVe application features beyond what is needed for DietPi compatibility.
- Upstreaming changes back to the original `xteve-project/xTeVe` repository (though the code should be kept compatible).

---

## 5. Codebase Audit Findings

The following issues were identified and **must** be fixed as part of this task:

### 5.1 Bugs

| Location | Issue | Severity |
|---|---|---|
| `src/maintenance.go:12` | `rand.Seed` is deprecated since Go 1.20; called in `InitMaintenance` | Medium |
| `src/maintenance.go:81` | `rand.Seed` called again redundantly in `randomTime` — double seeding | Medium |
| `src/internal/up2date/client/update.go:154` | `syscall.Exec` replaces the running process; systemd loses track of the PID, causing the service to appear "dead" on self-update | High |
| `src/toolchain.go:352–370` | `randomString` uses a non-cryptographically-random fallback (linear-congruential generator) when `crypto/rand` fails, but this fallback is never visible to callers and silently degrades security | Low |
| `go.mod:3–5` | `go 1.25.0` / `toolchain go1.26.1` are valid releases (Go 1.25: Aug 2025, Go 1.26: Feb 2026), but are **not available in Debian Bookworm apt** (`apt install golang` gives Go 1.19). The build pipeline must install the declared toolchain from go.dev rather than relying on apt. This is a build-environment concern, not a version-validity bug. | Medium (build pipeline) |

### 5.2 Magic Numbers

| Location | Value | Should Be |
|---|---|---|
| `src/config.go:52` | `System.PlexChannelLimit = 480` | Named constant `plexChannelLimit = 480` |
| `src/config.go:53` | `System.UnfilteredChannelLimit = 480` | Named constant `unfilteredChannelLimit = 480` |
| `src/config.go:54` | `System.Compatibility = "1.4.4"` | Named constant `minCompatibilityVersion = "1.4.4"` |
| `src/system.go:loadSettings` | default port `"34400"` | Named constant `defaultPort = "34400"` |
| `src/system.go:loadSettings` | `backup.keep = 10` | Named constant `defaultBackupKeep = 10` |
| `src/system.go:loadSettings` | `log.entries.ram = 500` | Named constant `defaultLogEntriesRAM = 500` |
| `src/system.go:loadSettings` | `buffer.size.kb = 1024` | Named constant `defaultBufferSizeKB = 1024` |
| `src/system.go:loadSettings` | `buffer.timeout = 500` | Named constant `defaultBufferTimeoutMS = 500` |
| `src/system.go:loadSettings` | `m3u8.adaptive.bandwidth.mbps = 10` | Named constant `defaultM3U8BandwidthMBPS = 10` |
| `src/system.go:loadSettings` | `mapping.first.channel = 1000` | Named constant `defaultMappingFirstChannel = 1000` |

### 5.3 Dead Code

| Location | Issue |
|---|---|
| `src/config.go:64` | Commented-out `System.Update.Git` assignment (old hardcoded URL) |
| `src/config.go:168` | Commented-out `System.Folder.Temp` UUID-path logic |
| `src/webserver.go` (buffer section) | Commented-out `Connection: keep-alive` header |

### 5.4 Deprecated / Problematic APIs

| Location | Issue |
|---|---|
| `src/maintenance.go` | `rand.Seed` (deprecated Go 1.20) — replace with `math/rand/v2` or use `rand.New(rand.NewSource(...))` per-caller |
| `src/internal/up2date/client/update.go` | `github.com/kardianos/osext` is deprecated; use `os.Executable()` (standard library since Go 1.8) |

### 5.5 Self-Update Mechanism (DietPi Incompatibility)

The `BinaryUpdate` function in `src/update.go` and `src/internal/up2date/client/update.go` downloads a new binary from GitHub and executes it via `syscall.Exec`. This is **incompatible** with DietPi's package management for the following reasons:

- Systemd tracks the process by PID; `syscall.Exec` changes the binary image but keeps the PID, causing subtle tracking issues on some kernel/systemd versions.
- DietPi manages software versions and updates via `dietpi-update` and `dietpi-software reinstall`; xTeVe's self-updater creates a parallel, untracked update path.
- Auto-updating a system binary without user awareness is poor practice on managed distributions.

**Requirement:** The self-update mechanism must be **disabled by default** when running under DietPi (detected by the presence of `/boot/dietpi` or the `DIETPI` environment variable set in the systemd unit). The update check can still run and log a notice, but automatic download and replacement must not occur. The feature flag `XteveAutoUpdate` in settings already exists and should default to `false` in DietPi mode.

---

## 6. Functional Requirements

### 6.1 DietPi Software Integration

- **FR-1.1**: xTeVe must be installable via `dietpi-software install <ID>` on DietPi.
- **FR-1.2**: xTeVe must be uninstallable via `dietpi-software uninstall <ID>` with clean removal of all installed files (binary, service, config symlinks).
- **FR-1.3**: The software must be listed in `dietpi-software` with name, description, category (Media Systems), and documentation URL.
- **FR-1.4**: xTeVe must be re-installable cleanly (reinstall must preserve user data in `/mnt/dietpi_userdata/xteve/`).
- **FR-1.5**: The install script must detect the CPU architecture (amd64, arm64, armv7) and install the appropriate pre-built binary.

### 6.2 System Service

- **FR-2.1**: A systemd unit file (`xteve.service`) must be provided that starts xTeVe as a background service.
- **FR-2.2**: The service must run as a dedicated unprivileged system user `xteve` (group `xteve`).
- **FR-2.3**: The service must be enabled at boot and started after installation.
- **FR-2.4**: The service must specify `Restart=on-failure` and `RestartSec=5s`.
- **FR-2.5**: The service must set `DIETPI=1` in the environment so that the application can detect it is running under DietPi.
- **FR-2.6**: The service must pass `--config /mnt/dietpi_userdata/xteve/` as the configuration directory.

### 6.3 User and Data Directory

- **FR-3.1**: The system user `xteve` must be created with home `/mnt/dietpi_userdata/xteve/`, shell `/usr/sbin/nologin`, and group membership in `video` and `render` (for hardware decode/encode access via FFmpeg).
- **FR-3.2**: User data (config, backups, cache, XEPG database) must be stored in `/mnt/dietpi_userdata/xteve/`.
- **FR-3.3**: The binary must be installed to `/usr/local/bin/xteve`.
- **FR-3.4**: Permissions on the data directory must be `xteve:xteve` with `0750`.

### 6.4 Build Pipeline

- **FR-4.1**: A `Makefile` must be provided that can build xTeVe for the host architecture (`make build`) and cross-compile for `linux/amd64`, `linux/arm64`, and `linux/arm` (`make build-all`).
- **FR-4.2**: The Makefile must embed the version string from `xteve.go` into the binary at build time.
- **FR-4.3**: Build artifacts must be placed in `build/` and excluded from version control (`.gitignore`).
- **FR-4.4**: A GitHub Actions workflow (`.github/workflows/build.yml`) must automatically build and publish release binaries when a git tag is pushed.

### 6.5 Code Quality Fixes

- **FR-5.1**: All magic numbers listed in §5.2 must be replaced with named constants in a new `src/constants.go` file.
- **FR-5.2**: All dead code listed in §5.3 must be removed.
- **FR-5.3**: Deprecated `rand.Seed` usage in `src/maintenance.go` must be replaced with the modern Go random API.
- **FR-5.4**: The `github.com/kardianos/osext` dependency must be replaced with `os.Executable()` from the standard library, and the dependency removed from `go.mod`/`go.sum`.
- **FR-5.5**: The build pipeline must ensure the Go toolchain declared in `go.mod` (`go 1.25.0` / `toolchain go1.26.1`) is available. Go 1.21+ automatically downloads the toolchain declared in `go.mod` when run with a sufficiently new `go` binary (via the built-in toolchain management introduced in Go 1.21). For CI (GitHub Actions), the `actions/setup-go` action with `go-version-file: go.mod` handles this automatically. For Makefile local builds, the Makefile must check that `go` ≥ 1.21 is present and print a clear error if not, directing the user to install from go.dev. The `go.mod` versions must not be downgraded.
- **FR-5.6**: The self-update mechanism must check for `DIETPI=1` environment variable and skip binary replacement, logging a notice instead.

### 6.6 Documentation

- **FR-6.1**: A `docs/dietpi-package.md` file must document the full DietPi integration: install/uninstall procedure, service management, data directory layout, default port, optional dependencies (FFmpeg, VLC).
- **FR-6.2**: A `docs/development.md` file must document how to build from source, run tests, the package structure, key architectural decisions, and how to extend the codebase.
- **FR-6.3**: All public-facing Go functions must have Go doc comments.
- **FR-6.4**: Configuration constants must be documented with their purpose, valid range, and default value.

---

## 7. Non-Functional Requirements

- **NFR-1**: The installed service must start in under 5 seconds on a Raspberry Pi 4 (ARM64).
- **NFR-2**: The binary size should remain under 30 MB. Measured baseline: ~14 MB unstripped on Linux amd64; expected ~10–12 MB with `-ldflags="-s -w"` (strip debug symbols). A stripped binary must be shipped in release builds.
- **NFR-3**: No root privileges may be assumed at runtime; all files must be writable by the `xteve` user.
- **NFR-4**: The package must function on Debian Bullseye and Bookworm (the base distributions for current DietPi versions).
- **NFR-5**: All changes must pass `go vet ./...` and `go build ./...` without errors or warnings.

---

## 8. Optional Dependencies

The following are optional runtime dependencies that should be documented. Note that `aSOFTWARE_DEPS` in `dietpi-software` installs **mandatory** dependencies automatically; since FFmpeg and VLC are genuinely optional (xTeVe runs without them using its built-in buffer), they must **not** be added to `aSOFTWARE_DEPS`. Instead, the install script should offer a user choice prompt (e.g., `G_WHIP_MENU`) to optionally install FFmpeg at install time, and document VLC as a separate manual install.

| Dependency | DietPi Software ID | Purpose |
|---|---|---|
| FFmpeg | 7 | Stream buffering/re-encoding (required only when `buffer = "ffmpeg"` in settings) |
| VLC | — | Alternative stream buffer (requires `cvlc` on PATH; no DietPi software ID — install via `apt install vlc`) |

---

## 9. Port Registration

xTeVe listens on TCP port **34400** by default (configurable). This port must be documented in the DietPi TCP/UDP port usage list.

---

## 10. DietPi Survey Integration

An entry for xTeVe (software ID + name) must be added to `.meta/dietpi-survey_report` in the main DietPi repository as part of the upstream PR. This file tracks which software titles participate in the anonymous install survey. This is distinct from the `G_DIETPI-SURVEY_SEND` bash function calls within `dietpi-software` itself, which handle runtime survey transmission.

---

## 11. Assumptions

- **A-1**: The xTeVe binary self-update will be disabled by default under DietPi; users who want to use it can enable `XteveAutoUpdate` in settings, understanding it bypasses DietPi package management.
- **A-2**: The DietPi software ID for xTeVe has not been assigned yet. On a live DietPi system, `dietpi-software free` prints the lowest unused ID (documented at [dietpi.com/docs](https://dietpi.com/docs/dietpi_tools/software_installation) and confirmed in the `dietpi-software` source). As a fallback, the implementor can inspect the `aSOFTWARE_NAME` array in the DietPi repository's `dietpi/dietpi-software` script to find an unused numeric ID manually.
- **A-3**: Pre-built binaries will be hosted in this repository's GitHub Releases (not the upstream `xteve-project/xTeVe-Downloads` repository) since this is a fork with independent changes.
- **A-4**: The minimum supported DietPi version is based on Debian Bullseye (DietPi v8.x+).
- **A-5**: The `go.mod` toolchain versions (`go 1.25.0` / `toolchain go1.26.1`) are valid stable releases (Go 1.25: August 2025, Go 1.26: February 2026) and must **not** be downgraded. They are simply unavailable via Debian apt; the build pipeline will install the correct toolchain directly from go.dev.

---

## 12. Out of Scope

- Web UI redesign or new UI features.
- HTTPS/TLS termination (recommended via reverse proxy, e.g., Nginx — DietPi software ID 83).
- Upstream pull request submission (this is a fork).
- Multi-instance support.
- Container/Docker packaging (separate concern from DietPi native packaging).
