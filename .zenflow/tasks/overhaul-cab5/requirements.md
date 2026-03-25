# Product Requirements Document — xTeVe Codebase Overhaul

## Overview

This PRD covers a full overhaul of the xTeVe codebase, targeting bugs, dead code, lazy patterns, magic numbers, and a focused improvement of the xTeVe-internal streaming buffer system to prevent infinite streaming loops and allow larger buffer sizes.

---

## 1. Bug Fixes

### 1.1 `defer` inside loops (critical — resource leak / panic risk)

**Files:** `src/buffer.go`, `src/toolchain.go`

**Problem:**
- In `bufferingStream` (Loop 2), `defer file.Close()` is placed inside a `for` loop. Defers are not scoped to loop iterations — they stack until the enclosing function returns, causing resource exhaustion.
- In `connectToStreamingServer`, `defer resp.Body.Close()` appears inside a `for` loop (including inside the `Redirect` goto block). The same stacking issue applies, worsened by the goto-redirect loop.
- In `thirdPartyBuffer`, `defer bufferFile.Close()` and `defer cmd.Wait()` are inside the function but after resources are already manually managed.
- In `toolchain.go → loadJSONFileToMap` and `readByteFromFile`: `defer f.Close()` is called before checking if `os.Open` returned an error. If the file open fails, `f` is nil and the deferred `Close()` will panic.

**Requirement:** Replace all `defer` calls inside loops with explicit `Close()` calls at the appropriate scope. In `toolchain.go`, guard `defer f.Close()` with an error check before deferring.

---

### 1.2 `panic(err)` in `thirdPartyBuffer` (critical)

**File:** `src/buffer.go` line ~1513

**Problem:** `panic(err)` is used when opening a file for append fails. A panic in a goroutine will crash the entire process. This should be an error path that calls `addErrorToStream` and returns gracefully.

**Requirement:** Replace `panic(err)` with proper error propagation (call `addErrorToStream`, log via `ShowError`, and `return`).

---

### 1.3 Dual file handle in `thirdPartyBuffer` (correctness bug)

**File:** `src/buffer.go` lines ~1610–1611

**Problem:** Inside the segment rotation loop, both `os.Create` and `os.OpenFile` are called on the same `tmpFile` in sequence, assigning both to `f`. The first file handle from `os.Create` is leaked.

**Requirement:** Use only `os.Create` to create and truncate the file (it returns a writable handle), then write directly. Remove the redundant `os.OpenFile` call.

---

### 1.4 Deprecated `http.CloseNotifier` in `bufferingStream`

**File:** `src/buffer.go` line ~267

**Problem:** `http.CloseNotifier` was deprecated in Go 1.11 and has been removed in newer Go versions. The interface type assertion `w.(http.CloseNotifier)` may fail silently or panic depending on the Go version.

**Requirement:** Replace `http.CloseNotifier` with `r.Context().Done()` to detect client disconnection. The streaming loop should select on context cancellation to detect when a client closes the connection.

---

### 1.5 `bufferingStream` Loop 2 — infinite loop risk (streaming loop bug)

**File:** `src/buffer.go` line ~263

**Problem:** Loop 2 (the client-send loop) has no exit condition if:
- The buffer goroutine has stopped but no error was recorded in `BufferClients`
- The stream folder exists but no new `.ts` files appear for an extended period
- The `http.CloseNotifier` path fails (see 1.4 above)

This can leave goroutines running indefinitely even when no client is connected, causing memory and goroutine leaks.

**Requirement:** Add an inactivity timeout to Loop 2. If no new segments are available for a configurable duration (e.g., `BufferTimeout * 10` milliseconds), the loop must call `killClientConnection` and return. Also add a check: if `BufferClients` no longer contains the stream entry, the loop must exit immediately.

---

### 1.6 Missing `rand.Read` error check in `randomString`

**File:** `src/toolchain.go` line ~354

**Problem:** `rand.Read` can theoretically return an error (on systems where the CSPRNG is unavailable). The error is silently discarded.

**Requirement:** Check the error returned by `rand.Read`. If an error occurs, fall back to a deterministic but unique value (e.g., time-based), or propagate the error.

---

### 1.7 Per-call `sync.RWMutex` in `showDebug`

**File:** `src/screen.go` line ~55

**Problem:** `showDebug` creates a new `sync.RWMutex{}` local to each call. This provides no mutual exclusion between concurrent callers — each call gets its own lock, meaning the protected code (appending to `WebScreenLog.Log`) is not actually synchronized.

**Requirement:** Move the debug-log mutex to a package-level variable shared across all log functions, or use the existing global `Lock` mutex.

---

### 1.8 JSON tag space in `RequestStruct.Settings.BufferSize`

**File:** `src/struct-webserver.go` line ~29

**Problem:** The JSON tag is `"buffer.size.kb, omitempty"` (space after comma). The Go `encoding/json` package is strict — this tag is parsed correctly in modern Go, but the leading space in the omitempty option is non-standard and fragile.

**Requirement:** Fix the tag to `"buffer.size.kb,omitempty"` (no space).

---

### 1.9 `ssdp.go` calls `os.Exit(0)` from a goroutine on SIGINT

**File:** `src/ssdp.go` line ~62

**Problem:** `os.Exit(0)` is called from within a goroutine on SIGINT signal. This bypasses any deferred cleanup in the main goroutine (open files, active streams, etc.).

**Requirement:** Replace `os.Exit(0)` with a mechanism that signals the main goroutine to shut down cleanly (e.g., a shared `context.Context` with cancel, or a dedicated shutdown channel).

---

## 2. Dead Code Removal

### 2.1 Commented-out code block in `struct-buffer.go`

**File:** `src/struct-buffer.go` lines 112–254

**Problem:** A large block of commented-out code remains in the file. It contains incomplete ffmpeg integration attempts, includes `os.Exit(0)` calls and debugging `fmt.Println` statements. It was never removed after development.

**Requirement:** Remove the entire commented-out block (lines 112–254).

---

### 2.2 Dead `Auto()` HTTP handler

**File:** `src/webserver.go` lines 188–211

**Problem:** The `Auto()` handler is registered in `StartWebserver` via a commented-out `http.HandleFunc` line, yet the handler function exists with a fully commented-out body and a `fmt.Println(channelID)` debug statement. It is non-functional dead code.

**Requirement:** Remove the `Auto()` function entirely. Remove the commented-out route registration line.

---

### 2.3 Commented-out `main()` in `authentication.go`

**File:** `src/internal/authentication/authentication.go` lines 47–~90

**Problem:** A `main()` function used for manual testing is left commented out. It contains `os.Exit(0)` calls and is dead code.

**Requirement:** Remove the commented-out `main()` function block.

---

## 3. Magic Numbers

### 3.1 Streaming constants in `buffer.go` and `bufferingStream`

**Problem:** The following magic numbers exist with no named constant or comment explaining their purpose:
- `timeOut > 200` → 200 × 100ms = 20-second startup timeout in Loop 1
- `n > 20` → max old `.ts` segments retained in the client send queue
- `i < 60` and `time.Duration(500)` → 30-second stream-limit video loop
- `len(m3u8Segments) > 30` / `m3u8Segments[15:]` → M3U8 segment deque thresholds
- `segment.Duration * 0.25` → HLS download timing headroom factor
- `sleep*1000` / `i + 100` → millisecond polling loop increments

**Requirement:** Extract these values into named constants at the top of `buffer.go` with explanatory names (e.g., `bufferStartupTimeoutIterations`, `maxRetainedSegments`, `streamLimitLoopSeconds`, `hlsTimingHeadroomFactor`, etc.).

---

### 3.2 Fixed timeout in `thirdPartyBuffer`

**Problem:** `timeout >= 20` (line ~1545) — 20 × 1000ms = 20s startup timeout — is a magic number.

**Requirement:** Extract as a named constant `thirdPartyBufferStartupTimeoutSec`.

---

## 4. Lazy / Low-Quality Code

### 4.1 `ioutil` deprecation

**Files:** `src/buffer.go`, `src/toolchain.go`, `src/xepg.go`, `src/backup.go`, `src/provider.go`, `src/update.go`, `src/webserver.go`, `src/html-build.go`, and internal packages

**Problem:** `io/ioutil` was deprecated in Go 1.16. All usages of `ioutil.ReadAll`, `ioutil.ReadDir`, `ioutil.WriteFile`, `ioutil.ReadFile` should be replaced with their `io` and `os` equivalents.

**Requirement:** Replace:
- `ioutil.ReadAll(r)` → `io.ReadAll(r)`
- `ioutil.ReadDir(dir)` → `os.ReadDir(dir)`
- `ioutil.WriteFile(f, d, m)` → `os.WriteFile(f, d, m)`
- `ioutil.ReadFile(f)` → `os.ReadFile(f)`

Remove `"io/ioutil"` imports.

---

### 4.2 `strings.Index` for contains-checks

**File:** `src/buffer.go`, `src/webserver.go`

**Problem:** `strings.Index(str, sub) != -1` is used instead of `strings.Contains(str, sub)`. This is less readable.

**Requirement:** Replace all `strings.Index(...) != -1` patterns with `strings.Contains(...)`.

---

### 4.3 `_ = i` pattern in loops

**File:** `src/buffer.go` (multiple locations)

**Problem:** `for i := ...; ...; ... { _ = i; ... }` — the loop variable `i` is declared but only used as `_ = i` to suppress the compiler warning. This indicates the loop body doesn't actually use the counter.

**Requirement:** Replace such loops with range over a slice or use `time.Sleep` in a simple `for` loop without a counter if the counter is truly unused. Where a count is needed, use the counter properly.

---

### 4.4 Error ignored in `getTmpFiles`

**File:** `src/buffer.go` line ~422

**Problem:** `strconv.ParseFloat` errors are silently swallowed. Non-numeric files in the temp folder will produce parse errors that are discarded without logging.

**Requirement:** Add a debug log when a file in the temp folder cannot be parsed as a float (e.g., if an unexpected file type appears), so the operator can diagnose issues.

---

## 5. Streaming Buffer Design — Fixes and Improvements

### 5.1 Prevent streaming loops (goroutine leak prevention)

**Problem:** The current buffer design has two nested loops in `bufferingStream`. Loop 2 can run indefinitely if the upstream buffer goroutine dies silently (no error written, no file removed). A client disconnecting while the server-side buffer is still running also leaves orphaned goroutines.

**Requirements:**
- Replace deprecated `http.CloseNotifier` with context-based disconnection detection (`r.Context().Done()`).
- Add a maximum inactivity duration to Loop 2: if no new segment files appear within `N` polling intervals (where `N` is derived from `BufferTimeout`), the loop must exit and call `killClientConnection`.
- Ensure `clientConnection` is checked at the top of every significant loop iteration in both `connectToStreamingServer` and `thirdPartyBuffer`.
- Add a goroutine count / active-stream logging mechanism at buffer entry and exit for diagnostics.

### 5.2 Expand choosable buffer sizes

**File:** `ts/settings_ts.ts` line ~348–349; compiled into `src/webUI.go`

**Current sizes:** 0.5 MB, 1 MB, 2 MB, 3 MB, 4 MB, 5 MB, 6 MB, 7 MB, 8 MB (max 8 MB)

**Problem:** For high-bitrate streams (4K, high-quality HEVC), 8 MB is insufficient to absorb network jitter. Operators need larger buffer options.

**Requirement:** Extend the buffer size options to include: 0.5 MB, 1 MB, 2 MB, 4 MB, 8 MB, 16 MB, 32 MB, 64 MB, 128 MB. Update both the `text` and `values` arrays in the settings TypeScript, and regenerate `webUI.go` accordingly.

---

## 6. Assumptions

- Go version in use supports `context`-based request cancellation (Go 1.7+) and `io` / `os` replacements for `ioutil` (Go 1.16+). The `go.mod` should be checked and updated if necessary.
- The TypeScript sources in `ts/` are compiled into `src/webUI.go` via `compileJS.sh`. The build step must be re-run after TypeScript changes.
- No external API contracts (Plex, Emby, HDHomeRun) are changed — only internal behavior is affected.
- Buffer size values stored in `settings.json` as KB integers remain the unit; only the UI choices and the max ceiling are expanded.
