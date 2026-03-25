# Technical Specification — xTeVe Codebase Overhaul

## 1. Technical Context

| Item | Value |
|------|-------|
| Language | Go 1.16 |
| UI | TypeScript (compiled to `src/webUI.go` via `ts/compileJS.sh`) |
| Dependencies | `github.com/gorilla/websocket v1.4.2`, `github.com/kardianos/osext`, `github.com/koron/go-ssdp v0.0.2` |
| Build | `go build` from repo root; UI rebuild: `cd ts && bash compileJS.sh` |
| Lint/Test | `go vet ./...` and `go build ./...` (no test suite exists beyond `src/internal/m3u-parser/`) |

---

## 2. Global Architecture

```
xteve.go          main()  →  src.Init() → src.StartSystem() → src.StartWebserver() [blocks]
src/config.go     global sync.Maps: BufferInformation, BufferClients; global Lock sync.RWMutex
src/buffer.go     bufferingStream → connectToStreamingServer / thirdPartyBuffer (goroutines)
src/screen.go     logging helpers (showInfo, showDebug, showWarning, ShowError)
src/toolchain.go  utility functions (file I/O, JSON, networking, randomString)
src/ssdp.go       SSDP/DLNA goroutine
src/webserver.go  HTTP handlers (StartWebserver blocks on ListenAndServe)
ts/settings_ts.ts UI settings panel (compiled into src/webUI.go)
```

---

## 3. Implementation Approach

Each subsection below corresponds to one category of fixes. All changes are confined to existing files; no new files are created.

---

### 3.1 `src/screen.go` — Fix per-call mutex (Bug 1.7)

**Problem confirmed (lines 55, 115, 130):** `showDebug`, `showWarning`, and `ShowError` each declare a local `var mutex = sync.RWMutex{}`. This provides zero inter-caller synchronisation.

**Fix:** Add one package-level variable:
```go
var screenLogMutex sync.RWMutex
```
Replace every `var mutex = sync.RWMutex{}` / `mutex.Lock()` / `mutex.Unlock()` in the three functions with `screenLogMutex.Lock()` / `screenLogMutex.Unlock()`. The existing package-level `Lock` in `config.go` is for buffer state and must **not** be reused for logging — keep them separate.

---

### 3.2 `src/toolchain.go` — Fix defer before error check + ioutil + rand.Read (Bugs 1.1, 1.6, 4.1)

**`loadJSONFileToMap` (line 236):** `defer f.Close()` is called unconditionally before the error from `os.Open` is checked. If `os.Open` fails, `f` is `nil` and the deferred `Close()` panics.

**Fix:**
```go
f, err := os.Open(getPlatformFile(file))
if err != nil {
    return
}
defer f.Close()
content, err := io.ReadAll(f)
```

**`readByteFromFile` (line 253):** Same pattern.

**Fix:**
```go
f, err := os.Open(getPlatformFile(file))
if err != nil {
    return
}
defer f.Close()
content, err = io.ReadAll(f)
```

**`checkFilePermission` (line 100):** Uses `ioutil.WriteFile` → replace with `os.WriteFile`.

**`saveMapToJSONFile` (line 225):** Uses `ioutil.WriteFile` → replace with `os.WriteFile`.

**`writeByteToFile` (line 264):** Uses `ioutil.WriteFile` → replace with `os.WriteFile`.

**`readStringFromFile` (line 279):** Uses `ioutil.ReadFile` → replace with `os.ReadFile`.

**`randomString` (line 354):** `rand.Read` error silently discarded.

**Fix:**
```go
func randomString(n int) string {
    const alphanum = "AB1CD2EF3GH4IJ5KL6MN7OP8QR9ST0UVWXYZ"
    bytes := make([]byte, n)
    if _, err := rand.Read(bytes); err != nil {
        // fallback: use time-based seed
        t := time.Now().UnixNano()
        for i := range bytes {
            bytes[i] = alphanum[int(t>>uint(i*3))%len(alphanum)]
        }
        return string(bytes)
    }
    for i, b := range bytes {
        bytes[i] = alphanum[b%byte(len(alphanum))]
    }
    return string(bytes)
}
```
Add `"time"` to imports; remove `"io/ioutil"`.

---

### 3.3 `src/struct-webserver.go` — Fix JSON tag space (Bug 1.8)

**Line 29:**
```go
// Before
BufferSize *int `json:"buffer.size.kb, omitempty"`
// After
BufferSize *int `json:"buffer.size.kb,omitempty"`
```

---

### 3.4 `src/ssdp.go` — Fix `os.Exit(0)` in goroutine (Bug 1.9)

**Problem (line 62):** `os.Exit(0)` called from a goroutine on SIGINT bypasses `main()`'s deferred cleanup and the HTTP server's graceful shutdown.

**Fix — two-part:**

**Part A — `src/config.go`:** Add a package-level shutdown channel:
```go
// ShutdownChan is closed to initiate a graceful shutdown.
var ShutdownChan = make(chan struct{})
```

**Part B — `src/ssdp.go`:** Replace `os.Exit(0)` with:
```go
case <-quit:
    adv.Bye()
    adv.Close()
    close(ShutdownChan)
    break loop
```
Remove `"os"` from imports if no longer used.

**Part C — `src/webserver.go` `StartWebserver()`:** Switch from `http.ListenAndServe` to `http.Server` with `Shutdown()`:
```go
srv := &http.Server{Addr: addr}
go func() {
    <-ShutdownChan
    srv.Shutdown(context.Background())
}()
err = srv.ListenAndServe()
if err == http.ErrServerClosed {
    err = nil
}
```
Add `"context"` import.

---

### 3.5 `src/struct-buffer.go` — Remove dead code (Dead code 2.1)

**Lines 112–254:** Entire commented-out block (two `/* ... */` comment regions containing incomplete ffmpeg integration, `os.Exit(0)` calls, and `fmt.Println` debug statements).

**Fix:** Delete lines 112–254 entirely. The file ends at line 111 after this removal.

---

### 3.6 `src/webserver.go` — Remove `Auto()` dead code (Dead code 2.2)

**Lines 188–211:** `Auto()` handler with a `fmt.Println` debug statement and a fully commented-out body.

**Fix:** Delete the entire `Auto()` function. Search for and remove any commented-out `http.HandleFunc` registration for `/auto/`. Remove `"fmt"` from imports if it becomes unused.

---

### 3.7 `src/internal/authentication/authentication.go` — Remove commented-out `main()` (Dead code 2.3)

**Lines 47–~130 (the `/* func main() { ... } */` block):** Remove the entire comment block. Also remove the commented-out `// "fmt"` and `// "log"` import lines if they are inside comments; if they suppress unused-import errors, they may be removed outright since the real imports don't use them.

---

### 3.8 `src/buffer.go` — Comprehensive fixes

This file has the most changes. They are grouped by type:

#### 3.8.1 Named constants (Magic numbers 3.1, 3.2)

Add a `const` block near the top of `buffer.go` (after the `import` block):

```go
const (
    // bufferStartupTimeoutIterations: max 100ms polling iterations before
    // declaring a stream startup timeout (~20 seconds).
    bufferStartupTimeoutIterations = 200

    // maxRetainedClientSegments: max number of .ts segment filenames kept in
    // the client's send queue before the oldest is pruned.
    maxRetainedClientSegments = 20

    // streamLimitLoopCount: number of 500ms iterations to play the
    // stream-limit video when the tuner limit is reached (~30 seconds).
    streamLimitLoopCount = 60

    // streamLimitSegmentSleepMS: milliseconds between stream-limit video writes.
    streamLimitSegmentSleepMS = 500

    // m3u8SegmentDequeMax: max M3U8 segment history before truncation.
    m3u8SegmentDequeMax = 30

    // m3u8SegmentDequeTrim: index to trim m3u8 segment history to.
    m3u8SegmentDequeTrim = 15

    // hlsTimingHeadroomFactor: fraction of segment duration reserved as
    // download headroom to prevent buffering stalls.
    hlsTimingHeadroomFactor = 0.25

    // hlsPollingIntervalMS: millisecond sleep between HLS polling iterations.
    hlsPollingIntervalMS = 100

    // hlsPollingStepMS: millisecond increment per HLS sleep loop step.
    hlsPollingStepMS = 100

    // thirdPartyBufferStartupTimeoutSec: seconds before declaring a
    // third-party buffer (ffmpeg/vlc) startup timeout.
    thirdPartyBufferStartupTimeoutSec = 20

    // bufferInactivityTimeoutIterations: max polling iterations with no new
    // .ts segments in Loop 2 before declaring an inactivity timeout and
    // killing the client connection.
    bufferInactivityTimeoutIterations = 300 // 300 × 100ms = 30s
)
```

Replace all corresponding literal values in the file with these constant names.

#### 3.8.2 Replace `http.CloseNotifier` with context (Bug 1.4)

**Signature change:** `bufferingStream` receives `r *http.Request`; use `r.Context()` directly — no signature change needed.

**Loop 2 (lines ~267–294):** Replace:
```go
cn, ok := w.(http.CloseNotifier)
if ok {
    select {
    case <-cn.CloseNotify():
        ...
    default:
        ...
    }
}
```
With:
```go
select {
case <-r.Context().Done():
    killClientConnection(streamID, playlistID, false)
    return
default:
    if c, ok := BufferClients.Load(playlistID + stream.MD5); ok {
        var clients = c.(ClientConnection)
        if clients.Error != nil {
            ShowError(clients.Error, 0)
            killClientConnection(streamID, playlistID, false)
            return
        }
    } else {
        return
    }
}
```

#### 3.8.3 Add inactivity timeout to Loop 2 (Bug 1.5)

Add a `lastSegmentTime` variable before Loop 2 begins, updated each time `tmpFiles` is non-empty. At the top of Loop 2, check if the elapsed time since `lastSegmentTime` exceeds `bufferInactivityTimeoutIterations × 100ms`:

```go
lastSegmentTime := time.Now()

for { // Loop 2
    // ... context check (3.8.2) ...

    // Inactivity timeout: exit if buffer goroutine is silent too long
    if time.Since(lastSegmentTime) > time.Duration(bufferInactivityTimeoutIterations)*100*time.Millisecond {
        showInfo("Streaming Status:Inactivity timeout — no new segments")
        killClientConnection(streamID, playlistID, false)
        return
    }

    // BufferClients entry missing → buffer goroutine exited
    if _, ok := BufferClients.Load(playlistID + stream.MD5); !ok {
        return
    }

    tmpFiles := getTmpFiles(&stream)
    if len(tmpFiles) > 0 {
        lastSegmentTime = time.Now()
    }
    // ... rest of loop ...
```

#### 3.8.4 Fix `defer file.Close()` inside Loop 2 (Bug 1.1)

**Lines ~316 and ~380:** `defer file.Close()` called inside the inner `for _, f := range tmpFiles` loop stacks defers until the enclosing function returns.

**Fix:** Replace every `defer file.Close()` inside that loop with explicit `file.Close()` immediately after use. Pattern:
```go
file, err := os.Open(fileName)
if err != nil {
    continue
}
// ... use file ...
file.Close()
```
Remove the extra `file.Close()` calls that were added as workarounds for the deferred close.

#### 3.8.5 Fix `defer resp.Body.Close()` inside `connectToStreamingServer` loops (Bug 1.1)

Multiple `defer resp.Body.Close()` calls appear inside the outer `for` loop and the `Redirect` goto block (lines ~707, 716, 727, 733, 737, 880, 891). Defers stack across the goto and loop iterations.

**Fix:**
- At the `Redirect` label block: call `resp.Body.Close()` explicitly before `goto Redirect`.
- In error paths that `return`: call `resp.Body.Close()` before `return`.
- Inside the TS read inner loop (line ~880): remove `defer resp.Body.Close()`.
- Inside the TS read inner loop (line ~891): remove `defer bufferFile.Close()`.
- Add a single `defer resp.Body.Close()` **after** the `Redirect:` label has been resolved and `resp` is valid — i.e., just after the successful `resp, err = client.Do(req)` path returns a non-redirect, non-error response. This is at the end of all redirect handling, before the `switch contentType` block.

Concretely, replace the scattered defers with one explicit placement:
```go
// After all redirect/error handling resolves, before switch contentType:
defer resp.Body.Close()
```
And ensure every early-return path calls `resp.Body.Close()` explicitly before returning.

#### 3.8.6 Fix `panic(err)` in `thirdPartyBuffer` (Bug 1.2)

**Line ~1513:**
```go
// Before
f, err = os.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
if err != nil {
    panic(err)
}

// After
f, err = os.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
if err != nil {
    ShowError(err, 0)
    addErrorToStream(err)
    cmd.Process.Kill()
    cmd.Wait()
    return
}
```

#### 3.8.7 Fix dual file handle in `thirdPartyBuffer` (Bug 1.3)

**Lines ~1609–1611:**
```go
// Before (leaks the handle from os.Create)
f, errCreate = os.Create(tmpFile)
f, errOpen = os.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0600)
if errCreate != nil || errOpen != nil { ... }

// After (create + truncate, then open for append)
f, err = os.Create(tmpFile)
if err != nil {
    cmd.Process.Kill()
    ShowError(err, 0)
    addErrorToStream(err)
    cmd.Wait()
    return
}
```
Remove `errCreate` and `errOpen` variables. The file returned by `os.Create` is writable — no second `OpenFile` is needed. If append-only semantics are required, use only `os.OpenFile` with `os.O_CREATE|os.O_TRUNC|os.O_WRONLY`.

#### 3.8.8 Replace `ioutil` (Bug 4.1)

- `ioutil.ReadDir(...)` → `os.ReadDir(...)` (line ~412 in `getTmpFiles`)
- `ioutil.ReadAll(resp.Body)` → `io.ReadAll(resp.Body)` (line ~810 in M3U8 case)

Remove `"io/ioutil"` from the import block (keep `"io"` — already imported).

#### 3.8.9 Replace `strings.Index` with `strings.Contains` (Bug 4.2)

**Line ~619:**
```go
// Before
if strings.Index(stream.URL, ".m3u8") != -1 {
// After
if strings.Contains(stream.URL, ".m3u8") {
```

#### 3.8.10 Replace `_ = i` loop patterns (Bug 4.3)

**Stream-limit loop (lines ~165–169):**
```go
// Before
for i := 1; i < 60; i++ {
    _ = i
    w.Write([]byte(content))
    time.Sleep(time.Duration(500) * time.Millisecond)
}

// After (use constant, no counter needed)
for range [streamLimitLoopCount]struct{}{} {
    w.Write([]byte(content))
    time.Sleep(streamLimitSegmentSleepMS * time.Millisecond)
}
```

**HLS sleep loops (lines ~1006–1015 and ~1277–1286):** Replace the `for i := 0.0; i < sleep*1000; i = i + 100 { _ = i; ... }` pattern with a context-aware ticker approach:

```go
deadline := time.Now().Add(time.Duration(sleep * float64(time.Second)))
for time.Now().Before(deadline) {
    time.Sleep(hlsPollingStepMS * time.Millisecond)
    if _, err := os.Stat(stream.Folder); os.IsNotExist(err) {
        break
    }
}
```

**`parseM3U8` loop (line ~1164):** `for i, line := range lines { _ = i; ... }` → `for _, line := range lines { ... }`.

#### 3.8.11 Add debug log for `strconv.ParseFloat` error in `getTmpFiles` (Bug 4.4)

**Lines ~423–427:**
```go
f, err := strconv.ParseFloat(fileID, 64)
if err == nil {
    fileIDs = append(fileIDs, f)
}
```
→
```go
f, err := strconv.ParseFloat(fileID, 64)
if err != nil {
    showDebug(fmt.Sprintf("Buffer Status:Skipping non-numeric temp file: %s", file.Name()), 2)
    continue
}
fileIDs = append(fileIDs, f)
```

#### 3.8.12 Add goroutine entry/exit logging

At the entry and exit of `connectToStreamingServer` and `thirdPartyBuffer`, add `showDebug` calls:
```go
// Entry
showDebug(fmt.Sprintf("Buffer Goroutine:START channel=%s playlist=%s", stream.ChannelName, playlistID), 1)
// Exit (defer at top of function)
defer showDebug(fmt.Sprintf("Buffer Goroutine:END channel=%s playlist=%s", stream.ChannelName, playlistID), 1)
```

---

### 3.9 `src/webserver.go` — Replace `ioutil` and `strings.Index` (Bugs 4.1, 4.2)

- Replace all `ioutil.ReadAll` → `io.ReadAll`, `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile`, `ioutil.ReadDir` → `os.ReadDir`.
- Replace `strings.Index(...) != -1` patterns with `strings.Contains(...)`.
- Remove `"io/ioutil"` import; ensure `"io"` and `"os"` are present.

---

### 3.10 Other files using `ioutil` (Bug 4.1)

The following files contain `ioutil` usage; apply the same mechanical replacements and remove `"io/ioutil"` imports:

| File | Replacement needed |
|------|--------------------|
| `src/xepg.go` | `ioutil.ReadAll` → `io.ReadAll`, `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile` |
| `src/backup.go` | `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile` |
| `src/provider.go` | `ioutil.ReadAll` → `io.ReadAll`, `ioutil.ReadFile` → `os.ReadFile` |
| `src/update.go` | `ioutil.ReadAll` → `io.ReadAll` |
| `src/html-build.go` | `ioutil.ReadFile` → `os.ReadFile` |
| `src/internal/authentication/authentication.go` | `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile` |
| `src/internal/m3u-parser/m3u-parser_test.go` | `ioutil.ReadFile` → `os.ReadFile` |
| `src/internal/up2date/client/client.go` | `ioutil.ReadAll` → `io.ReadAll` |
| `src/internal/imgcache/cache.go` | All `ioutil` variants → `io`/`os` equivalents |

---

### 3.11 `ts/settings_ts.ts` — Expand buffer size options (Requirement 5.2)

**Lines 348–349:**
```typescript
// Before
var text:any[]   = ["0.5 MB", "1 MB", "2 MB", "3 MB", "4 MB", "5 MB", "6 MB", "7 MB", "8 MB"]
var values:any[] = ["512", "1024", "2048", "3072", "4096", "5120", "6144", "7168", "8192"]

// After
var text:any[]   = ["0.5 MB", "1 MB", "2 MB", "4 MB", "8 MB", "16 MB", "32 MB", "64 MB", "128 MB"]
var values:any[] = ["512", "1024", "2048", "4096", "8192", "16384", "32768", "65536", "131072"]
```

After editing the TypeScript, regenerate `src/webUI.go` by running:
```bash
cd ts && bash compileJS.sh
```

---

## 4. Source Code Structure Changes

| File | Change Type | Summary |
|------|-------------|---------|
| `src/screen.go` | Refactor | Add `screenLogMutex`; fix per-call mutex in 3 functions |
| `src/toolchain.go` | Bug fix + refactor | Guard `defer f.Close()` with error check; replace `ioutil`; fix `randomString` |
| `src/struct-webserver.go` | Bug fix | Remove space in JSON tag |
| `src/config.go` | Feature | Add `ShutdownChan chan struct{}` |
| `src/ssdp.go` | Bug fix | Replace `os.Exit(0)` with `close(ShutdownChan)` |
| `src/webserver.go` | Bug fix + refactor | Switch to `http.Server` with graceful shutdown; remove `Auto()`; replace `ioutil`/`strings.Index` |
| `src/struct-buffer.go` | Dead code | Remove commented-out lines 112–254 |
| `src/buffer.go` | Bug fix + refactor | Named constants; context-based disconnect; inactivity timeout; fix all defers-in-loops; fix panic; fix dual handle; replace ioutil/strings patterns; replace `_ = i` loops; add goroutine logging |
| `src/internal/authentication/authentication.go` | Dead code + refactor | Remove commented `main()`; replace `ioutil` |
| `src/xepg.go` | Refactor | Replace `ioutil` |
| `src/backup.go` | Refactor | Replace `ioutil` |
| `src/provider.go` | Refactor | Replace `ioutil` |
| `src/update.go` | Refactor | Replace `ioutil` |
| `src/html-build.go` | Refactor | Replace `ioutil` |
| `src/internal/m3u-parser/m3u-parser_test.go` | Refactor | Replace `ioutil` |
| `src/internal/up2date/client/client.go` | Refactor | Replace `ioutil` |
| `src/internal/imgcache/cache.go` | Refactor | Replace `ioutil` |
| `ts/settings_ts.ts` | Feature | Expand buffer size options |
| `src/webUI.go` | Generated | Regenerated from TypeScript after settings change |

---

## 5. Data Model / API / Interface Changes

- **Buffer size values** remain KB integers in `settings.json`. The existing range of valid values is extended; no migration is needed (old values remain valid). Values above 8192 are new options.
- **No external API changes** (Plex/Emby/HDHomeRun device XML, M3U/XMLTV endpoints unchanged).
- **WebSocket settings protocol** (`struct-webserver.go`): The JSON tag fix (`buffer.size.kb,omitempty` without space) is technically a bug fix; modern Go parses both forms identically, so no client-side breakage.

---

## 6. Verification Approach

```bash
# Full build (catches all type errors and missing imports)
go build ./...

# Vet (catches defer misuse, printf format errors, etc.)
go vet ./...

# Run existing unit test
go test ./src/internal/m3u-parser/...

# TypeScript compilation check
cd ts && bash compileJS.sh

# Manual streaming smoke test:
# 1. Start xTeVe with buffer=xteve
# 2. Open an HLS stream in VLC
# 3. Kill VLC mid-stream → verify Loop 2 exits cleanly (context cancellation)
# 4. Leave stream idle → verify inactivity timeout fires after ~30s
# 5. Test tuner-limit video plays ~30 seconds then ends
```
