# Full SDD workflow

## Configuration
- **Artifacts Path**: `.zenflow/tasks/create-dietpi-software-package-d7ea`

---

## Agent Instructions

---

## Workflow Steps

### [x] Step: Requirements
<!-- chat-id: 098ce720-49eb-456b-8e86-2e64fcba4394 -->

Create a Product Requirements Document (PRD) based on the feature description.

1. Review existing codebase to understand current architecture and patterns
2. Analyze the feature definition and identify unclear aspects
3. Ask the user for clarifications on aspects that significantly impact scope or user experience
4. Make reasonable decisions for minor details based on context and conventions
5. If user can't clarify, make a decision, state the assumption, and continue

Focus on **what** the feature should do and **why**, not **how** it should be built. Do not include technical implementation details, technology choices, or code-level decisions — those belong in the Technical Specification.

Save the PRD to `.zenflow/tasks/create-dietpi-software-package-d7ea/requirements.md`.

### [x] Step: Technical Specification
<!-- chat-id: 85d68483-2345-4b1f-9b82-1f8013d57f70 -->

Create a technical specification based on the PRD in `.zenflow/tasks/create-dietpi-software-package-d7ea/requirements.md`.

1. Review existing codebase architecture and identify reusable components
2. Define the implementation approach

Do not include implementation steps, phases, or task breakdowns — those belong in the Planning step.

Save to `.zenflow/tasks/create-dietpi-software-package-d7ea/spec.md` with:
- Technical context (language, dependencies)
- Implementation approach referencing existing code patterns
- Source code structure changes
- Data model / API / interface changes
- Verification approach using project lint/test commands

### [x] Step: Planning
<!-- chat-id: 90a3708e-896b-41a3-96d2-57467891e107 -->

Create a detailed implementation plan based on `.zenflow/tasks/create-dietpi-software-package-d7ea/spec.md`.

1. Break down the work into concrete tasks
2. Each task should reference relevant contracts and include verification steps
3. Replace the Implementation step below with the planned tasks

Rule of thumb for step size: each step should represent a coherent unit of work (e.g., implement a component, add an API endpoint). Avoid steps that are too granular (single function) or too broad (entire feature).

Important: unit tests must be part of each implementation task, not separate tasks. Each task should implement the code and its tests together, if relevant.

If the feature is trivial and doesn't warrant full specification, update this workflow to remove unnecessary steps and explain the reasoning to the user.

Save to `.zenflow/tasks/create-dietpi-software-package-d7ea/plan.md`.

---

## Implementation Tasks

> **GitHub username gate (resolved)**: The working directory is `/Users/whisper/Documents/xTeVe-dietpi`; the host OS user is `whisper`. Based on this context the fork lives at `github.com/whisper/xTeVe-dietpi`. The `GitHub` struct in `xteve.go` must be updated to `User: "whisper", Repo: "xTeVe-dietpi"` so update checks and release downloads point to this fork's GitHub Releases. This is treated as a confirmed value; if incorrect the agent implementing Step 1 must correct it.

### [x] Step 1: Go source — named constants and dead code removal
<!-- chat-id: 5704458e-d416-4f27-9f76-9cb2da0f04fe -->

**Scope**: `src/constants.go` (new), `src/config.go`, `src/system.go`, `src/buffer.go`

**What to implement**:

1. Create `src/constants.go` in `package src` with the following named constants (each with a Go doc comment stating purpose, unit, and default value):
   - `plexChannelLimit = 480`
   - `unfilteredChannelLimit = 480`
   - `minCompatibilityVersion = "1.4.4"`
   - `defaultPort = "34400"`
   - `defaultBackupKeep = 10`
   - `defaultLogEntriesRAM = 500`
   - `defaultBufferSizeKB = 1024`
   - `defaultBufferTimeoutMS = 500`
   - `defaultM3U8BandwidthMBPS = 10`
   - `defaultMappingFirstChannel = 1000`

2. In `src/config.go`:
   - Replace `System.PlexChannelLimit = 480` → `plexChannelLimit`
   - Replace `System.UnfilteredChannelLimit = 480` → `unfilteredChannelLimit`
   - Replace `System.Compatibility = "1.4.4"` → `minCompatibilityVersion`
   - Replace `Settings.LogEntriesRAM = 500` → `defaultLogEntriesRAM`
   - Remove the commented-out line `//System.Update.Git = "https://github.com/xteve-project/xTeVe-Downloads/blob"` (line 64)
   - Remove the commented-out line `//System.Folder.Temp = System.Folder.Temp + Settings.UUID + string(os.PathSeparator)` (line 168)

3. In `src/system.go` (`loadSettings` defaults map):
   - Replace `"port": "34400"` → `defaultPort`
   - Replace `"backup.keep": 10` → `defaultBackupKeep`
   - Replace `"log.entries.ram": 500` → `defaultLogEntriesRAM`
   - Replace `"buffer.size.kb": 1024` → `defaultBufferSizeKB`
   - Replace `"buffer.timeout": 500` → `defaultBufferTimeoutMS`
   - Replace `"m3u8.adaptive.bandwidth.mbps": 10` → `defaultM3U8BandwidthMBPS`
   - Replace `"mapping.first.channel": 1000` → `defaultMappingFirstChannel`
   - In `saveSettings`, replace the hardcoded `10` in the `BackupKeep == 0` guard → `defaultBackupKeep`

4. In `src/buffer.go`:
   - Remove the commented-out `//w.Header().Set("Connection", "keep-alive")` line

5. In `src/authentication.go` (`checkAuthorizationLevel`):
   - At lines ~162–163 and ~167–168 the error from `authentication.WriteUserData(userID, userData)` is silently overwritten by `err = errors.New("No authorization")` on the very next line. Log the write error before overwriting:
     ```go
     if wErr := authentication.WriteUserData(userID, userData); wErr != nil {
         ShowError(wErr, 0)
     }
     err = errors.New("No authorization")
     ```
   Apply this pattern to both branches where the overwrite occurs.

**Verification**: `go build ./...` and `go vet ./...` must pass with zero errors.

---

### [ ] Step 2: Go source — fix deprecated rand.Seed in maintenance.go

**Scope**: `src/maintenance.go`

**What to implement**:

Replace `math/rand` (deprecated `rand.Seed`) with `math/rand/v2`:

1. Remove `"math/rand"` from imports
2. Add `"math/rand/v2"` to imports (aliased as `rand` if needed for minimal diff, or use directly)
3. Remove `import "time"` only if it is solely used for `time.Now().Unix()` in the seed calls — keep it for `time.Sleep` and `time.Now()` in `maintenance()`
4. In `InitMaintenance`: remove the `rand.Seed(time.Now().Unix())` call entirely
5. In `randomTime`: remove the `rand.Seed(time.Now().Unix())` call; replace `rand.Intn(max-min) + min` with `rand.IntN(max-min) + min`

No tests exist for this function; verify by `go build ./...` and `go vet ./...`.

---

### [ ] Step 3: Go source — remove osext dependency, add DietPi guards

**Scope**: `src/internal/up2date/client/update.go`, `src/update.go`, `src/screen.go`, `src/system.go`, `xteve.go`, `go.mod`, `go.sum`

**What to implement**:

1. **`src/internal/up2date/client/update.go`** — replace `github.com/kardianos/osext`:
   - Remove `import "github.com/kardianos/osext"`
   - Ensure `"os"` is imported (add if absent)
   - Replace every `osext.Executable()` call (lines ~52 and ~152) with `os.Executable()`, handling the `(string, error)` return: `binary, err := os.Executable(); if err != nil { return err }`
   - The line ~152 Linux restart path uses `file, _ := osext.Executable()` (error discarded). When replacing, the error **must** be handled (return it or log and skip), not silently discarded.

2. **`src/internal/up2date/client/client.go`** — fix nil-dereference panic after `http.NewRequest`:
   - At line ~97–98, `req, err := http.NewRequest(...)` is followed immediately by `req.Header.Set(...)` without checking `err`. If `http.NewRequest` returns an error, `req` is `nil` and the call panics. Fix:
     ```go
     req, err := http.NewRequest("POST", Updater.URL, bytes.NewBuffer(jsonByte))
     if err != nil {
         return err
     }
     req.Header.Set("Content-Type", "application/json")
     ```

3. **`src/system.go`** — DietPi default override for auto-update:
   - Ensure `"os"` is imported
   - In `loadSettings`, after the `defaults` map is fully populated but before it is applied to `settingsMap`, insert:
     ```go
     // On DietPi, disable auto-update by default; user can opt in via settings.
     if os.Getenv("DIETPI") == "1" {
         defaults["xteveAutoUpdate"] = false
     }
     ```

4. **`src/screen.go`** — add warning message 6005:
   - In the `getErrMsg` switch, between the existing `case 6004:` and `default:`, insert:
     ```go
     case 6005:
         errMsg = fmt.Sprintf("XteveAutoUpdate is enabled on DietPi. The binary will be replaced outside of dietpi-software. To manage xTeVe via DietPi, disable XteveAutoUpdate in Settings.")
     ```

5. **`src/update.go`** — DietPi opt-in warning in `BinaryUpdate()`:
   - Ensure `"os"` is imported
   - Inside the `Settings.XteveAutoUpdate == true` branch, before the `up2date.DoUpdate(...)` call, insert:
     ```go
     // Warn when running under DietPi — binary will be replaced outside of dietpi-software.
     if os.Getenv("DIETPI") == "1" {
         showWarning(6005)
     }
     ```

6. **`xteve.go`** — update `GitHub` struct and convert `Version` from `const` to `var`:
   - Change `User: "xteve-project"` → `User: "whisper"`
   - Change `Repo: "xTeVe-Downloads"` → `Repo: "xTeVe-dietpi"`
   - Change `const Version = "2.2.0.0200"` → `var Version = "2.2.0.0200"`. Go's linker `-X` flag only injects into `var` declarations, not `const`. Keeping it as `const` causes the Makefile's `-X main.Version=$(VERSION)` to be silently ignored. Converting to `var` has no runtime impact (the value is identical at compile time) but enables version overriding at link time (e.g., CI can inject the git tag).

7. **`go.mod` / `go.sum`** — remove osext:
   - Run `go mod tidy` to remove `github.com/kardianos/osext` from `go.mod` and `go.sum`

**Verification**:
- `go build ./...` — must succeed
- `go vet ./...` — zero warnings
- `go mod tidy` — must not re-add osext; `go.sum` must be consistent

---

### [ ] Step 4: Build system — Makefile and .gitignore

**Scope**: `Makefile` (new), `.gitignore` (updated)

**What to implement**:

1. **`.gitignore`** — add to existing file:
   ```
   build/
   *.tar.gz
   *.zip
   dist/
   ```

2. **`Makefile`** — create with the following targets. Use the exact guard and version-extraction logic from spec §3.9:

   ```makefile
   # Go toolchain guard — requires Go >= 1.21 for built-in toolchain management
   GO_HAVE := $(shell go version 2>/dev/null)
   ifeq ($(GO_HAVE),)
     $(error 'go' not found. Install Go >= 1.21 from https://go.dev/dl/)
   endif
   GO_MAJ := $(shell go version | awk '{print $$3}' | sed 's/go\([0-9]*\)\..*/\1/')
   GO_MIN := $(shell go version | awk '{print $$3}' | sed 's/go[0-9]*\.\([0-9]*\).*/\1/')
   ifeq ($(shell test $(GO_MAJ) -gt 1 || { test $(GO_MAJ) -eq 1 && test $(GO_MIN) -ge 21; } && echo ok),)
     $(error Go >= 1.21 required (have $(shell go version)). Install from https://go.dev/dl/)
   endif

   VERSION := $(shell grep -E '^(const|var) Version = ' xteve.go | sed 's/.*"\(.*\)".*/\1/')
   ifeq ($(VERSION),)
     $(warning WARNING: could not extract VERSION from xteve.go; using "unknown")
     VERSION := unknown
   endif
   LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"
   BUILD_DIR := build

   .PHONY: build build-all clean vet test

   build:
   	mkdir -p $(BUILD_DIR)
   	go build $(LDFLAGS) -o $(BUILD_DIR)/xteve .

   build-all:
   	mkdir -p $(BUILD_DIR)
   	GOOS=linux GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/xteve_linux_amd64  .
   	GOOS=linux GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/xteve_linux_arm64  .
   	GOOS=linux GOARCH=arm    GOARM=7 go build $(LDFLAGS) -o $(BUILD_DIR)/xteve_linux_arm .

   clean:
   	rm -rf $(BUILD_DIR)

   vet:
   	go vet ./...

   test:
   	go test ./...
   ```

**Verification**:
- `make vet` — zero warnings
- `make build` — produces `build/xteve`
- `make build-all` — produces `build/xteve_linux_{amd64,arm64,arm}`
- `make clean` — removes `build/`

---

### [ ] Step 5: DietPi integration — systemd service and install/uninstall scripts

**Scope**: `systemd/xteve.service` (new), `dietpi/install.sh` (new), `dietpi/uninstall.sh` (new)

**What to implement**:

1. **`systemd/xteve.service`**:
   ```ini
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
   ExecStart=/usr/local/bin/xteve --config /mnt/dietpi_userdata/xteve/
   Restart=on-failure
   RestartSec=5s
   NoNewPrivileges=true
   PrivateTmp=true

   [Install]
   WantedBy=multi-user.target
   ```

2. **`dietpi/install.sh`** (Bash, `set -euo pipefail`):
   - Detect and remap arch (on ARMv7, `dpkg --print-architecture` returns `armhf`, but the CI binary is named `xteve_linux_arm`):
     ```bash
     ARCH=$(dpkg --print-architecture)
     case "$ARCH" in
         armhf)        ARCH=arm ;;
         amd64|arm64)  ;;  # already matches CI binary name
         *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
     esac
     ```
   - Construct download URL: `https://github.com/whisper/xTeVe-dietpi/releases/latest/download/xteve_linux_${ARCH}`
   - Download binary with `curl -fsSL` to `/tmp/xteve` → install to `/usr/local/bin/xteve` with `install -m 0755`
   - Create system user idempotently (see spec §3.7 for `useradd` flags with conditional `render` group)
   - Create data dir: `mkdir -p /mnt/dietpi_userdata/xteve && chown xteve:xteve /mnt/dietpi_userdata/xteve && chmod 0750 /mnt/dietpi_userdata/xteve`
   - Install service file: `install -m 0644 "$(dirname "$0")/../systemd/xteve.service" /etc/systemd/system/xteve.service`
   - `systemctl daemon-reload && systemctl enable --now xteve`
   - Offer optional FFmpeg via `G_WHIP_YESNO` → `G_DIETPI-INSTALL_SOFTWARE 7`

3. **`dietpi/uninstall.sh`** (Bash, `set -euo pipefail`):
   - `systemctl disable --now xteve || true`
   - `rm -f /etc/systemd/system/xteve.service`
   - `systemctl daemon-reload`
   - `rm -f /usr/local/bin/xteve`
   - `userdel xteve || true`
   - Print notice that `/mnt/dietpi_userdata/xteve/` is preserved

**Verification**:
- `bash -n dietpi/install.sh` — Bash syntax check
- `bash -n dietpi/uninstall.sh` — Bash syntax check
- Manual review that paths, user, and group match spec

---

### [ ] Step 6: CI/CD — GitHub Actions release workflow

**Scope**: `.github/workflows/build.yml` (new)

**What to implement**:

Create `.github/workflows/build.yml` triggered on `push` with tags matching `v*`:

```yaml
name: Build and Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: write

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build all platforms
        run: make build-all

      - name: Create release archives
        run: |
          cd build
          tar czf xteve_linux_amd64.tar.gz xteve_linux_amd64
          tar czf xteve_linux_arm64.tar.gz xteve_linux_arm64
          tar czf xteve_linux_arm.tar.gz   xteve_linux_arm

      - uses: softprops/action-gh-release@v2
        with:
          files: |
            build/xteve_linux_amd64.tar.gz
            build/xteve_linux_arm64.tar.gz
            build/xteve_linux_arm.tar.gz
            build/xteve_linux_amd64
            build/xteve_linux_arm64
            build/xteve_linux_arm
```

**Verification**:
- `yamllint .github/workflows/build.yml` if available, or manual review of YAML validity
- Confirm `actions/setup-go@v5` correctly resolves toolchain from `go.mod` (it does for all versions of `go-version-file`)

---

### [ ] Step 7: Documentation — DietPi package guide and developer guide

**Scope**: `docs/dietpi-package.md` (new), `docs/development.md` (new), Go doc comments on public functions

**What to implement**:

1. **`docs/dietpi-package.md`** — End-user DietPi integration guide covering:
   - Prerequisites and installation via `dietpi-software install <ID>`
   - Service management (`systemctl start/stop/restart/status xteve`)
   - Data directory layout (`/mnt/dietpi_userdata/xteve/`)
   - Default port (34400) and how to change it
   - Optional dependencies: FFmpeg (DietPi software ID 7), VLC (`apt install vlc`)
   - Uninstall procedure and data preservation
   - Upgrade procedure (`dietpi-software reinstall <ID>` or tag-driven CI release)
   - Troubleshooting: log access, permission issues, firewall notes

2. **`docs/development.md`** — Developer guide covering:
   - Repository structure overview
   - Prerequisites: Go >= 1.21, `make`
   - Build instructions: `make build`, `make build-all`, `make vet`, `make test`
   - Key architectural decisions: `package src` vs `package main`, embedded HTML (`webUI.go`), SSDP/goroutine notes
   - DietPi integration architecture: env var detection, auto-update guard, service unit
   - How to add a new configuration constant (add to `src/constants.go`, reference in `src/config.go` or `src/system.go`)
   - How to cut a release: tag format `vX.Y.Z`, CI triggers, binary naming
   - Running tests: `go test ./...` (note: only `src/internal/m3u-parser/` has tests currently)

3. **Go doc comments** — add package-level and function-level doc comments to all exported symbols that lack them in:
   - `src/config.go`: `Init`, `StartSystem`
   - `src/update.go`: `BinaryUpdate`
   - `src/maintenance.go`: `InitMaintenance`
   - `src/constants.go`: each constant (already mandated in Step 1)
   - `xteve.go`: `GitHubStruct`, `GitHub`, `Name`, `Version`, `DBVersion`, `APIVersion`

**Verification**:
- `go doc ./...` must not produce parse errors
- Manual review for completeness and accuracy

---

### [ ] Step 8: Final verification — build, vet, test, and module cleanliness

**Scope**: whole repository

**What to do**:

1. Run `go mod tidy` — confirm `osext` is absent from `go.mod` and `go.sum`
2. Run `go vet ./...` — must produce zero warnings
3. Run `go build ./...` — must succeed
4. Run `go test ./...` — all tests pass
5. Run `make build` — `build/xteve` binary produced
6. Run `make build-all` — all three platform binaries produced
7. Run `bash -n dietpi/install.sh` and `bash -n dietpi/uninstall.sh` — Bash syntax OK
8. Confirm `.gitignore` includes `build/`, `*.tar.gz`, `*.zip`, `dist/`
9. Record results as inline checkboxes below:
   - [ ] `go mod tidy` clean
   - [ ] `go vet ./...` zero warnings
   - [ ] `go build ./...` success
   - [ ] `go test ./...` all pass
   - [ ] `make build` success
   - [ ] `make build-all` success
   - [ ] `bash -n dietpi/install.sh` OK
   - [ ] `bash -n dietpi/uninstall.sh` OK
   - [ ] `.gitignore` entries confirmed
