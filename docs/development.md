# xTeVe — Developer Guide

This guide covers building from source, understanding the architecture, adding new features, and cutting releases.

---

## Prerequisites

| Tool | Minimum Version | Notes |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.21 | Required for built-in toolchain management and `math/rand/v2` |
| `make` | any | GNU Make or compatible |
| `git` | any | For cloning and tagging releases |

The `go.mod` declares `go 1.25.0` with `toolchain go1.26.1`. Go's built-in toolchain management (≥ 1.21) will download and use the declared toolchain version automatically. A host Go ≥ 1.21 is sufficient.

---

## Repository Structure

```
xTeVe-dietpi/
├── xteve.go                     # package main — entry point, CLI flags, version constants
├── go.mod / go.sum              # Module definition and checksums
├── Makefile                     # Build targets: build, build-all, vet, test, clean
├── .gitignore
│
├── src/                         # package src — all library code
│   ├── constants.go             # Named constants (no magic numbers)
│   ├── config.go                # Init() and StartSystem() — top-level wiring
│   ├── system.go                # loadSettings(), saveSettings(), system bootstrap
│   ├── update.go                # BinaryUpdate() — self-updater with DietPi guard
│   ├── maintenance.go           # InitMaintenance() — scheduler (backup, EPG refresh, update)
│   ├── buffer.go                # Streaming proxy / buffer
│   ├── webserver.go             # HTTP server, routes, graceful shutdown
│   ├── webUI.go                 # Embedded web UI (generated — do not edit directly)
│   ├── xepg.go                  # XEPG channel database builder
│   ├── authentication.go        # Session and API token handling
│   ├── backup.go                # Backup and restore logic
│   ├── data.go                  # Data structures and JSON persistence helpers
│   ├── m3u.go                   # M3U playlist parser
│   ├── hdhr.go                  # HDHomeRun emulation (device discovery)
│   ├── provider.go              # Provider data fetching (M3U, XMLTV, HDHomeRun)
│   ├── ssdp.go                  # SSDP/UPnP advertisement (DietPi → Plex discovery)
│   ├── screen.go                # Console log formatting and error messages
│   ├── toolchain.go             # Utility functions (file I/O, paths, JSON helpers)
│   ├── struct-*.go              # Go struct definitions (system, webserver, XML, etc.)
│   └── internal/
│       ├── m3u-parser/          # M3U tokenizer (has unit tests)
│       └── up2date/
│           └── client/          # Self-updater HTTP client
│               ├── client.go    # Version check request
│               └── update.go    # Binary download and hot-swap (uses os.Executable())
│
├── html/                        # Web UI source (HTML, CSS, JS, images)
│   └── ...
├── ts/                          # TypeScript source for the web UI JS bundle
│   └── ...
│
├── systemd/
│   └── xteve.service            # systemd unit file for DietPi
├── dietpi/
│   ├── install.sh               # DietPi install script (run as subprocess by dietpi-software)
│   └── uninstall.sh             # DietPi uninstall script
├── .github/
│   └── workflows/
│       └── build.yml            # CI: cross-compile and publish on tag push
└── docs/
    ├── dietpi-package.md        # End-user DietPi guide
    └── development.md           # This file
```

---

## Build Instructions

### Build for the host architecture

```bash
make build
# Produces: build/xteve
```

### Cross-compile for all supported Linux targets

```bash
make build-all
# Produces:
#   build/xteve_linux_amd64   (x86-64)
#   build/xteve_linux_arm64   (AArch64)
#   build/xteve_linux_arm     (ARMv7 / armhf)
```

### Run static analysis

```bash
make vet
```

### Run tests

```bash
make test
# Currently only src/internal/m3u-parser/ has unit tests.
```

### Clean build artefacts

```bash
make clean
```

### Version injection

The Makefile extracts the version string from `xteve.go` at build time:

```makefile
VERSION := $(shell grep -E '^(const|var) Version = ' xteve.go | sed 's/.*"\(.*\)".*/\1/')
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"
```

`Version` is declared as `var` (not `const`) so the Go linker's `-X` flag can override it. CI injects the git tag (e.g., `v2.3.0`) at link time.

---

## Key Architectural Decisions

### `package main` vs `package src`

`xteve.go` is the only `package main` file. It parses CLI flags, populates the global `src.System` struct, and calls into `src` for all application logic. The entire library lives in `package src` under `src/`.

### Embedded web UI (`src/webUI.go`)

The HTML, CSS, JavaScript, and image assets under `html/` are embedded in the binary via `src/webUI.go`. This file is **generated** — do not edit it directly. To rebuild the web UI after modifying files in `html/` or compiling TypeScript in `ts/`:

```bash
# Start xTeVe in developer mode — it serves live files from disk instead of the embedded copy
./build/xteve -dev -config /tmp/xteve-dev/
```

In dev mode, `src/html-build.go` runs `BuildGoFile()` once at startup to regenerate `src/webUI.go` from the files in `html/`.

### Global state

`src/config.go` declares package-level globals (`System`, `Settings`, `Data`, `WebScreenLog`, `BufferInformation`, `BufferClients`, `Lock`). These are intentional — xTeVe is a single-process server and the globals are protected by `sync.RWMutex` and `sync.Map` where needed.

### SSDP / goroutines

`src/ssdp.go` registers xTeVe as an HDHomeRun device via SSDP so Plex and Emby can discover it automatically on the local network. The SSDP alive announcements run in a background goroutine. The maintenance scheduler (`src/maintenance.go`) also runs in its own goroutine.

### DietPi integration

DietPi-specific behaviour is controlled by a single environment variable:

```
DIETPI=1
```

This is set in `systemd/xteve.service` and is never set in non-DietPi environments. Guards that check `os.Getenv("DIETPI") == "1"` are the canonical way to vary behaviour for DietPi.

Current DietPi-specific behaviours:

| Behaviour | Where implemented |
|---|---|
| `XteveAutoUpdate` defaults to `false` | `src/system.go` → `loadSettings()` |
| Warning logged when auto-update runs on DietPi | `src/update.go` → `BinaryUpdate()` (warning 6005) |

---

## Adding a New Configuration Constant

Magic numbers must not appear in the source. All numeric and string defaults live in `src/constants.go`.

**Steps:**

1. Open `src/constants.go` and add a new `const` with a Go doc comment:
   ```go
   // defaultFooBar is the default value for the FooBar setting, in <unit>.
   // Default value: 42.
   defaultFooBar = 42
   ```

2. Reference the constant wherever the literal previously appeared (typically in `src/config.go` or the `defaults` map in `src/system.go`).

3. Run `go vet ./...` and `go build ./...` to confirm there are no errors.

---

## DietPi Integration Architecture

```
dietpi-software install <ID>
        │
        ▼
dietpi/install.sh          ← shell script (Bash, set -euo pipefail)
        │
        ├─ detect arch (dpkg --print-architecture → amd64 | arm64 | arm)
        ├─ download binary from GitHub Releases (curl -fsSL)
        ├─ install binary → /usr/local/bin/xteve
        ├─ create system user xteve
        ├─ create data dir /mnt/dietpi_userdata/xteve/
        ├─ install systemd/xteve.service → /etc/systemd/system/xteve.service
        └─ systemctl enable --now xteve

systemd/xteve.service
        │
        ├─ User=xteve, Group=xteve
        ├─ Environment="DIETPI=1"          ← triggers DietPi-specific behaviour in Go
        ├─ ExecStart=/usr/local/bin/xteve --config /mnt/dietpi_userdata/xteve/
        └─ Restart=on-failure
```

The Go binary detects DietPi via `os.Getenv("DIETPI") == "1"`. No compile-time build tags are used, which means the same binary works on both DietPi and non-DietPi Linux systems.

---

## Cutting a Release

1. Ensure all changes are committed and pushed to `master` (or your working branch).

2. Tag the release using the `vX.Y.Z` format:
   ```bash
   git tag v2.3.0
   git push origin v2.3.0
   ```

3. GitHub Actions (`.github/workflows/build.yml`) will:
   - Check out the tag.
   - Run `make build-all` (cross-compiles for `amd64`, `arm64`, `arm`).
   - Create `.tar.gz` archives for each binary.
   - Publish a GitHub Release with the binaries and archives as assets.

4. Asset naming convention (must match `dietpi/install.sh`):
   ```
   xteve_linux_amd64
   xteve_linux_arm64
   xteve_linux_arm
   ```

5. Update `var Version` in `xteve.go` to match the new tag before tagging if you want the embedded version string to match the release tag:
   ```go
   var Version = "2.3.0.0200"
   ```
   The Makefile's `-X main.Version=$(VERSION)` will inject the extracted version at link time; CI may also inject the git tag directly.

---

## Running Tests

```bash
go test ./...
```

Only `src/internal/m3u-parser/` has unit tests currently. When adding new functionality, add tests alongside the implementation in the same package.

---

## Dependency Management

All dependencies are declared in `go.mod`. To add a new dependency:

```bash
go get github.com/example/library@v1.2.3
go mod tidy
```

To remove an unused dependency:

```bash
go mod tidy
```

Current direct dependencies:

| Module | Purpose |
|---|---|
| `github.com/gorilla/websocket` | WebSocket server for the web UI real-time log and status updates |
| `github.com/koron/go-ssdp` | SSDP/UPnP advertisement for HDHomeRun device discovery |

The previously used `github.com/kardianos/osext` has been removed; `os.Executable()` from the standard library is used in its place.
