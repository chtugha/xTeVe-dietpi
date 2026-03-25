# Technical Specification: xTeVe DietPi Software Package

## 1. Technical Context

### Language & Runtime
- **Language**: Go (module path `xteve`, `package main` entry point in `xteve.go`, library code in `package src` under `src/`)
- **Go toolchain**: `go 1.25.0` / `toolchain go1.26.1` as declared in `go.mod` — not downgraded; installed via Go's built-in toolchain management (≥ Go 1.21 host required) or `actions/setup-go` in CI
- **Target platforms**: `linux/amd64`, `linux/arm64`, `linux/arm` (ARMv7)

### Current Dependencies (`go.mod`)
| Module | Version | Action |
|---|---|---|
| `github.com/gorilla/websocket` | v1.4.2 | Keep |
| `github.com/kardianos/osext` | v0.0.0-20190222173326-2bc1f35cddc0 | **Remove** — replaced by `os.Executable()` |
| `github.com/koron/go-ssdp` | v0.0.2 | Keep |
| `golang.org/x/net` | v0.52.0 | Keep (indirect) |
| `golang.org/x/sys` | v0.42.0 | Keep (indirect) |

### DietPi Environment
- Base OS: Debian Bullseye / Bookworm
- Service manager: systemd
- User data root: `/mnt/dietpi_userdata/`
- Binary location: `/usr/local/bin/xteve`
- Service user: `xteve` (system account, no login shell)
- DietPi detection: `DIETPI=1` environment variable (set in systemd unit)

---

## 2. Source Code Structure Changes

### New Files
```
src/constants.go                          # All named constants (replaces magic numbers)
dietpi/                                   # DietPi integration scripts
  install.sh                             # Called by dietpi-software to install xTeVe
  uninstall.sh                           # Called by dietpi-software to uninstall xTeVe
systemd/
  xteve.service                          # systemd unit file
Makefile                                 # Build targets: build, build-all, clean, vet
.github/
  workflows/
    build.yml                            # CI: build + release binaries on git tag push
docs/
  dietpi-package.md                      # DietPi integration guide (end-user)
  development.md                         # Developer guide (build, test, architecture)
```

### Modified Files
| File | Changes |
|---|---|
| `.gitignore` | Add `build/`, `*.zip`, `*.tar.gz` |
| `go.mod` | Remove `github.com/kardianos/osext` |
| `go.sum` | Remove osext entry (via `go mod tidy`) |
| `src/constants.go` | **New** — named constants extracted from `src/config.go` and `src/system.go` |
| `src/config.go` | Replace inline literals with constants from `src/constants.go`; remove commented-out dead code (lines 64, 168) |
| `src/maintenance.go` | Replace `rand.Seed` + `math/rand` with `math/rand/v2` global functions (no seeding needed) |
| `src/update.go` | Add `DIETPI` env-var check: skip `DoUpdate` call, log notice instead |
| `src/internal/up2date/client/update.go` | Replace `github.com/kardianos/osext` with `os.Executable()`; remove `osext` import |
| `src/webserver.go` | Remove commented-out `Connection: keep-alive` header |
| `xteve.go` | Update `GitHub` var to point to this fork's repo for update downloads |

---

## 3. Implementation Approach

### 3.1 `src/constants.go` — Named Constants

Create a new file `src/constants.go` in `package src` exporting all numeric/string defaults as package-level `const` declarations. Reference existing usages in `src/config.go` (lines 52–54) and `src/system.go` (`loadSettings` defaults map).

```go
package src

const (
    // plexChannelLimit is the maximum number of channels exposed to Plex DVR.
    plexChannelLimit = 480

    // unfilteredChannelLimit is the maximum number of channels in the
    // unfiltered (raw provider) channel list.
    unfilteredChannelLimit = 480

    // minCompatibilityVersion is the oldest settings.json version that can be
    // migrated automatically. Databases older than this require a fresh install.
    minCompatibilityVersion = "1.4.4"

    // defaultPort is the TCP port xTeVe listens on when none is configured.
    defaultPort = "34400"

    // defaultBackupKeep is the number of automatic backups retained.
    defaultBackupKeep = 10

    // defaultLogEntriesRAM is the number of log lines kept in memory for the
    // web UI log view.
    defaultLogEntriesRAM = 500

    // defaultBufferSizeKB is the stream buffer size in kilobytes.
    defaultBufferSizeKB = 1024

    // defaultBufferTimeoutMS is the time in milliseconds before a buffer
    // connection is considered timed out.
    defaultBufferTimeoutMS = 500

    // defaultM3U8BandwidthMBPS is the adaptive bitrate selection ceiling used
    // when parsing HLS (M3U8) playlists.
    defaultM3U8BandwidthMBPS = 10

    // defaultMappingFirstChannel is the lowest channel number assigned during
    // initial XEPG channel mapping.
    defaultMappingFirstChannel = 1000
)
```

Replace the literal values in `src/config.go`:
- `System.PlexChannelLimit = 480` → `System.PlexChannelLimit = plexChannelLimit`
- `System.UnfilteredChannelLimit = 480` → `System.UnfilteredChannelLimit = unfilteredChannelLimit`
- `System.Compatibility = "1.4.4"` → `System.Compatibility = minCompatibilityVersion`

Replace the literal values in `src/system.go` `defaults` map:
- `"backup.keep": 10` → `defaultBackupKeep`
- `"log.entries.ram": 500` → `defaultLogEntriesRAM`
- `"buffer.size.kb": 1024` → `defaultBufferSizeKB`
- `"buffer.timeout": 500` → `defaultBufferTimeoutMS`
- `"m3u8.adaptive.bandwidth.mbps": 10` → `defaultM3U8BandwidthMBPS`
- `"mapping.first.channel": 1000` → `defaultMappingFirstChannel`
- `"port": "34400"` → `defaultPort`

Also replace in `src/system.go` `saveSettings`:
- `settings.BackupKeep == 0` guard resets to `10` → `defaultBackupKeep`

### 3.2 Dead Code Removal

**`src/config.go` line 64** — Remove the commented-out line:
```go
//System.Update.Git = "https://github.com/xteve-project/xTeVe-Downloads/blob"
```

**`src/config.go` line 168** — Remove the commented-out block:
```go
//System.Folder.Temp = System.Folder.Temp + Settings.UUID + string(os.PathSeparator)
```

**`src/webserver.go`** — Locate and remove any commented-out `Connection: keep-alive` header line in the buffer/stream handler section.

> **Pre-implementation check**: The sibling `overhaul-cab5` task (all steps `[x]`) already touched `src/webserver.go` for graceful shutdown. Before applying this removal, verify with `grep -n "keep-alive" src/webserver.go` whether the dead code is still present. If it was already removed by that task, skip this sub-step.

### 3.3 `src/maintenance.go` — Deprecated `rand.Seed`

Replace the `math/rand` import with `math/rand/v2`. In Go 1.20+, the global `rand` functions use an automatically-seeded source; `rand.Seed` is a no-op deprecated call.

`math/rand/v2` removes `Seed` entirely and provides `rand.IntN(n)` (replaces `rand.Intn(n)`).

Changes:
- Remove `import "math/rand"` and `import "time"` (if only used for seeding)
- Add `import "math/rand/v2"`
- In `InitMaintenance`: remove `rand.Seed(time.Now().Unix())`
- In `randomTime`: remove `rand.Seed(time.Now().Unix())`; replace `rand.Intn(max-min) + min` with `rand.IntN(max-min) + min`
- Keep `import "time"` for `time.Sleep` and `time.Now()` in `maintenance()`

### 3.4 `src/internal/up2date/client/update.go` — Remove `osext` Dependency

Replace `github.com/kardianos/osext` with `os.Executable()`:
- Remove `import "github.com/kardianos/osext"`
- Add `import "os"` (already present)
- Replace `osext.Executable()` calls (lines 52, 152) with `os.Executable()`
- `os.Executable()` returns `(string, error)`; handle the error: `binary, err := os.Executable(); if err != nil { return err }`

After this change, run `go mod tidy` to remove `osext` from `go.mod` and `go.sum`.

### 3.5 `src/update.go` — DietPi Auto-Update Guard

Requirements §A-1 states that users may opt in to the self-updater on DietPi by explicitly enabling `XteveAutoUpdate` in settings, understanding they bypass DietPi package management. The guard must therefore respect this explicit opt-in.

The correct behaviour, reconciling FR-5.6 and A-1:

| `XteveAutoUpdate` | `DIETPI=1` | Result |
|---|---|---|
| `false` (default) | any | Skip update, log notice (existing path via `showWarning(6004)`) |
| `true` | not set | Perform update (existing path) |
| `true` | `1` | Log a prominent DietPi warning, then proceed with update (user explicitly opted in) |

Implementation: in `BinaryUpdate()`, inside the `Settings.XteveAutoUpdate == true` branch, **before** calling `up2date.DoUpdate(...)`:

```go
// Warn when running under DietPi — user has explicitly opted in.
if os.Getenv("DIETPI") == "1" {
    showWarning(6005) // new warning: self-update on DietPi bypasses package management
}
// DoUpdate proceeds regardless — explicit opt-in is honoured.
```

Add warning message `6005` to `src/screen.go` (the `getWarningMsg` / `showWarning` lookup):
```
6005: "XteveAutoUpdate is enabled on DietPi. The binary will be replaced outside of dietpi-software. To manage xTeVe via DietPi, disable XteveAutoUpdate in Settings."
```

This preserves the update path for opt-in users while making the risk visible, and the default (`XteveAutoUpdate = false`) remains safe on DietPi with no code change needed there.

### 3.6 `systemd/xteve.service` — Service Unit File

```ini
[Unit]
Description=xTeVe M3U Proxy for Plex DVR and Emby Live TV
Documentation=https://github.com/REPLACE_WITH_GITHUB_USERNAME/xTeVe-dietpi
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=xteve
Group=xteve
Environment="DIETPI=1"
ExecStart=/usr/local/bin/xteve --config /mnt/dietpi_userdata/xteve/
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### 3.7 `dietpi/install.sh` — Install Script

The DietPi install script is sourced by `dietpi-software`. It must:

1. Detect CPU architecture using `dpkg --print-architecture` → map to `amd64 | arm64 | armhf`
2. Download the appropriate pre-built binary from this repo's GitHub Releases
3. Install binary to `/usr/local/bin/xteve` with mode `0755`
4. Create system user `xteve` (idempotent: skip if user exists):
   ```bash
   # render group is only present when GPU drivers are installed; add conditionally
   EXTRA_GROUPS=''
   getent group render &>/dev/null && EXTRA_GROUPS=',render'
   id -u xteve &>/dev/null || \
       useradd --system \
               --home-dir /mnt/dietpi_userdata/xteve \
               --shell /usr/sbin/nologin \
               --groups "video${EXTRA_GROUPS}" \
               xteve
   ```
5. Create and `chown xteve:xteve` the data directory `/mnt/dietpi_userdata/xteve/` with mode `0750`
6. Install systemd unit from `systemd/xteve.service` to `/etc/systemd/system/xteve.service`
7. `systemctl daemon-reload && systemctl enable --now xteve`
8. Offer optional FFmpeg install via `G_WHIP_YESNO` — if accepted, install using the DietPi helper:
   ```bash
   G_WHIP_YESNO 'Would you like to install FFmpeg for stream buffering/re-encoding?\n\nThis is only required when "Buffer" is set to "FFmpeg" in xTeVe settings.' \
       && G_DIETPI-INSTALL_SOFTWARE 7
   ```
   `G_DIETPI-INSTALL_SOFTWARE` is the correct in-script idiom for installing another DietPi software title from within a sourced install script; calling `dietpi-software install` recursively is not supported.

### 3.8 `dietpi/uninstall.sh` — Uninstall Script

1. `systemctl disable --now xteve`
2. `rm -f /etc/systemd/system/xteve.service`
3. `systemctl daemon-reload`
4. `rm -f /usr/local/bin/xteve`
5. User data under `/mnt/dietpi_userdata/xteve/` is **preserved** (DietPi convention)
6. System user `xteve` is removed: `userdel xteve`

### 3.9 `Makefile` — Build System

Targets:

| Target | Action |
|---|---|
| `build` | Build for host arch with version ldflags, output `build/xteve` |
| `build-all` | Cross-compile for all three targets, output `build/xteve_linux_{amd64,arm64,arm}` |
| `clean` | Remove `build/` directory |
| `vet` | Run `go vet ./...` |
| `test` | Run `go test ./...` |

Version string embedded via:
```makefile
VERSION := $(shell grep '^const Version = ' xteve.go | sed 's/.*"\(.*\)".*/\1/')
ifeq ($(VERSION),)
  $(warning WARNING: could not extract VERSION from xteve.go; using "unknown")
  VERSION := unknown
endif
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"
```

The `grep` pattern is anchored to `^const Version = ` (matches the exact declaration line) so it is resilient to comment changes or other variables that happen to contain the word "Version".

Go toolchain guard (at top of Makefile):
```makefile
GO_HAVE := $(shell go version 2>/dev/null)
ifeq ($(GO_HAVE),)
  $(error 'go' not found. Install Go >= 1.21 from https://go.dev/dl/)
endif
# Extract major.minor as two integers for numeric comparison
GO_MAJ := $(shell go version | awk '{print $$3}' | sed 's/go\([0-9]*\)\..*/\1/')
GO_MIN := $(shell go version | awk '{print $$3}' | sed 's/go[0-9]*\.\([0-9]*\).*/\1/')
ifeq ($(shell test $(GO_MAJ) -gt 1 || { test $(GO_MAJ) -eq 1 && test $(GO_MIN) -ge 21; } && echo ok),)
  $(error Go >= 1.21 required (have $(shell go version)). Install from https://go.dev/dl/)
endif
```

### 3.10 `.github/workflows/build.yml` — CI/CD

Trigger: `push` with tag matching `v*`.

Steps:
1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version-file: go.mod` (automatically resolves toolchain)
3. Build for all three targets with stripped ldflags
4. Create archives: `xteve_linux_amd64.tar.gz`, etc.
5. `softprops/action-gh-release@v2` to publish as GitHub Release assets

### 3.11 `.gitignore` Updates

Add to existing `.gitignore`:
```
build/
*.tar.gz
*.zip
dist/
```

### 3.12 `xteve.go` — Fork GitHub Reference

Update the `GitHub` variable so update checks and binary downloads reference this fork's releases (not the upstream `xteve-project/xTeVe-Downloads`).

> **Planning gate — resolve before implementing this section**: The `User` and `Repo` values in the `GitHub` struct determine where the binary updater fetches release metadata and archives. These must be the real GitHub username and repository name that will host the release assets produced by the CI workflow (§3.10). If left as placeholders, the binary will attempt to check a non-existent repo on every startup, logging 404 errors.
>
> **Action for the Planning step**: Confirm the GitHub username and the releases repository name and substitute them here before writing the code. Until confirmed, the placeholder values below must not be committed:

```go
var GitHub = GitHubStruct{
    Branch: "master",
    User:   "REPLACE_WITH_GITHUB_USERNAME",   // e.g. "whisper"
    Repo:   "REPLACE_WITH_RELEASES_REPO",     // repo where release ZIPs are published
    Update: true,
}
```

The CI workflow (§3.10) will publish release assets to the same repository as the source (`xTeVe-dietpi`), so `Repo` will typically be `"xTeVe-dietpi"` or a dedicated `"xTeVe-dietpi-downloads"` repo — this must be confirmed before implementation.

---

## 4. Data Model / API / Interface Changes

No changes to the HTTP API, WebSocket protocol, settings JSON schema, or XEPG/M3U data formats.

The only runtime-observable behaviour changes are:

- **Auto-update on DietPi** — governed by the §3.5 decision table:
  - `XteveAutoUpdate=false` (default) + `DIETPI=1`: update is skipped; existing warning `6004` is emitted. No binary is downloaded. This is the same path as non-DietPi with auto-update disabled; no code change required for this row.
  - `XteveAutoUpdate=true` + `DIETPI=1`: warning `6005` is emitted and the update proceeds normally — binary is downloaded and the process is replaced via `syscall.Exec`. The user has explicitly opted in and accepted the consequences.
  - `XteveAutoUpdate=true` + `DIETPI` unset: update proceeds unchanged (existing behaviour).
- **All configuration defaults** are semantically identical to before; only the source (named constant vs. inline literal) changes.

---

## 5. File Layout After Implementation

```
xTeVe-dietpi/
├── .github/
│   └── workflows/
│       └── build.yml
├── .gitignore                 # updated
├── dietpi/
│   ├── install.sh
│   └── uninstall.sh
├── docs/
│   ├── development.md
│   └── dietpi-package.md
├── go.mod                     # osext removed
├── go.sum                     # osext removed
├── Makefile
├── src/
│   ├── constants.go           # new
│   ├── config.go              # dead code removed, literals → constants
│   ├── maintenance.go         # math/rand/v2, no rand.Seed
│   ├── system.go              # literals → constants
│   ├── update.go              # DIETPI env guard
│   ├── webserver.go           # commented-out header removed
│   ├── internal/
│   │   └── up2date/
│   │       └── client/
│   │           └── update.go  # osext → os.Executable()
│   └── ... (unchanged)
├── systemd/
│   └── xteve.service
└── xteve.go                   # GitHub var updated to fork
```

---

## 6. Verification Approach

### Automated (CI)
- `go vet ./...` — must pass with zero warnings
- `go build ./...` — must succeed for all three target platforms
- `go test ./...` — all test packages must pass (currently only `src/internal/m3u-parser/` contains tests)

### Manual (Local)
```bash
make vet         # go vet ./...
make build       # produces build/xteve
make build-all   # produces three platform binaries
```

### DietPi Integration Test (on target hardware or QEMU)
1. Run `dietpi/install.sh` on a DietPi Bookworm image → service starts, web UI reachable at `http://<ip>:34400/web/`
2. Run `dietpi/uninstall.sh` → binary and service removed, `/mnt/dietpi_userdata/xteve/` preserved
3. **Auto-update default path** (`XteveAutoUpdate=false`): with `DIETPI=1` set (as in the service unit), trigger `BinaryUpdate()` → confirm warning `6004` is emitted and no binary is downloaded or replaced.
4. **Auto-update opt-in path** (`XteveAutoUpdate=true`): with `DIETPI=1` set, trigger `BinaryUpdate()` when a newer version is available → confirm warning `6005` is emitted and the update proceeds (binary is replaced, process restarts).

### Go Module Cleanliness
```bash
go mod tidy      # must not re-add osext; go.sum must remain consistent
```
