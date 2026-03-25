# Full SDD workflow

## Configuration
- **Artifacts Path**: {@artifacts_path} → `.zenflow/tasks/{task_id}`

---

## Agent Instructions

---

## Workflow Steps

### [x] Step: Requirements
<!-- chat-id: c99f3bdd-e0de-45de-990b-d09c49f2f77d -->

Create a Product Requirements Document (PRD) based on the feature description.

1. Review existing codebase to understand current architecture and patterns
2. Analyze the feature definition and identify unclear aspects
3. Ask the user for clarifications on aspects that significantly impact scope or user experience
4. Make reasonable decisions for minor details based on context and conventions
5. If user can't clarify, make a decision, state the assumption, and continue

Focus on **what** the feature should do and **why**, not **how** it should be built. Do not include technical implementation details, technology choices, or code-level decisions — those belong in the Technical Specification.

Save the PRD to `{@artifacts_path}/requirements.md`.

### [x] Step: Technical Specification
<!-- chat-id: 2b520ca4-fc7f-4266-9b59-d9aa6b85dcba -->

Create a technical specification based on the PRD in `{@artifacts_path}/requirements.md`.

1. Review existing codebase architecture and identify reusable components
2. Define the implementation approach

Do not include implementation steps, phases, or task breakdowns — those belong in the Planning step.

Save to `{@artifacts_path}/spec.md` with:
- Technical context (language, dependencies)
- Implementation approach referencing existing code patterns
- Source code structure changes
- Data model / API / interface changes
- Verification approach using project lint/test commands

### [x] Step: Planning
<!-- chat-id: 818c5901-ee25-49bc-b843-4d8ee45bdee7 -->

Create a detailed implementation plan based on `{@artifacts_path}/spec.md`.

### [x] Step: Trivial fixes and dead code removal
<!-- chat-id: fb455ae0-1154-4dd8-9901-d459d595e912 -->

Apply the smallest, self-contained fixes across three files. These have no interdependencies.

**`src/struct-webserver.go`** (spec 3.3 / bug 1.8):
- Fix JSON tag space: change `"buffer.size.kb, omitempty"` → `"buffer.size.kb,omitempty"` (line ~29)

**`src/struct-buffer.go`** (spec 3.5 / dead code 2.1):
- Delete the entire commented-out block on lines 112–254 (incomplete ffmpeg integration attempt with `os.Exit`, `fmt.Println` debug stubs)

**`src/internal/authentication/authentication.go`** (spec 3.7 / dead code 2.3):
- Remove the commented-out `/* func main() { ... } */` block (~lines 47–130)
- Remove any commented-out `// "fmt"` / `// "log"` import lines inside that block

Verification: `go build ./...` and `go vet ./...` must pass after each file change.

### [x] Step: Fix shared mutex in `src/screen.go`
<!-- chat-id: 2690e407-3d6a-441b-8bfe-dba0f16ca79f -->

Spec reference: 3.1 / bug 1.7.

- Add a package-level `var screenLogMutex sync.RWMutex` variable at the top of `src/screen.go`
- In `showDebug`, `showWarning`, and `ShowError`: remove the local `var mutex = sync.RWMutex{}` declaration and replace all `mutex.Lock()` / `mutex.Unlock()` calls with `screenLogMutex.Lock()` / `screenLogMutex.Unlock()`
- Ensure `"sync"` remains imported

Verification: `go build ./...` and `go vet ./...` pass.

### [x] Step: Fix `src/toolchain.go` — defer guards, ioutil, randomString
<!-- chat-id: 7feaa4a5-2bb8-4043-9e13-d119b9b386c1 -->

Spec reference: 3.2 / bugs 1.1, 1.6, 4.1.

- **`loadJSONFileToMap`**: move `defer f.Close()` to after the `os.Open` error check (currently panics on nil `f` if open fails)
- **`readByteFromFile`**: same defer-guard fix
- **`checkFilePermission`**: `ioutil.WriteFile` → `os.WriteFile`
- **`saveMapToJSONFile`**: `ioutil.WriteFile` → `os.WriteFile`
- **`writeByteToFile`**: `ioutil.WriteFile` → `os.WriteFile`
- **`readStringFromFile`**: `ioutil.ReadFile` → `os.ReadFile`
- **`randomString`**: check error from `rand.Read`; on error fall back to a time-based (`time.Now().UnixNano()`) deterministic value instead of silently using corrupt bytes
- Remove `"io/ioutil"` from imports; add `"time"` if not already present

Verification: `go build ./...` and `go vet ./...` pass.

### [x] Step: Graceful shutdown — `src/config.go`, `src/ssdp.go`, `src/webserver.go`
<!-- chat-id: d9f658cb-4b20-4241-b647-d7145a8a87d4 -->

Spec reference: 3.4 / bug 1.9. Also covers removal of the `Auto()` dead-code handler (spec 3.6 / dead code 2.2).

**`src/config.go`**:
- Add `var ShutdownChan = make(chan struct{})` as a package-level variable

**`src/ssdp.go`**:
- Replace `os.Exit(0)` (called in the SIGINT goroutine) with `close(ShutdownChan)` followed by a clean break from the signal loop
- Call `adv.Bye()` and `adv.Close()` before closing the channel
- Remove `"os"` import if it becomes unused

**`src/webserver.go`**:
- Switch from bare `http.ListenAndServe` to `http.Server` with a `Shutdown()` goroutine that waits on `<-ShutdownChan`; treat `http.ErrServerClosed` as a non-error
- Add `"context"` import
- Remove the `Auto()` function (lines ~188–211: function body is fully commented out and has a stray `fmt.Println(channelID)` debug statement)
- Remove the commented-out `http.HandleFunc` route registration for `/auto/`
- Remove `"fmt"` from imports if it becomes unused after Auto() removal

Verification: `go build ./...` and `go vet ./...` pass.

### [x] Step: `src/buffer.go` — Part A: Named constants and simple cleanups
<!-- chat-id: 8113354d-3b40-41c1-bdc1-f160b8c7eb6c -->

Spec reference: 3.8.1, 3.8.8, 3.8.9, 3.8.10, 3.8.11, 3.8.12.

**Named constants** (add a `const` block near the top of `buffer.go` after imports):
- `bufferStartupTimeoutIterations = 200` (20s startup timeout in Loop 1)
- `maxRetainedClientSegments = 20` (max .ts segments in client send queue)
- `streamLimitLoopCount = 60` (iterations of tuner-limit video loop)
- `streamLimitSegmentSleepMS = 500` (ms between limit-video writes)
- `m3u8SegmentDequeMax = 30` (max M3U8 segment history)
- `m3u8SegmentDequeTrim = 15` (trim index for M3U8 segment deque)
- `hlsTimingHeadroomFactor = 0.25` (HLS download headroom fraction)
- `hlsPollingIntervalMS = 100` (ms between HLS polling ticks)
- `hlsPollingStepMS = 100` (ms step per HLS sleep sub-loop)
- `thirdPartyBufferStartupTimeoutSec = 20` (3rd-party buffer startup timeout)
- `bufferInactivityTimeoutIterations = 300` (300 × 100ms = 30s inactivity timeout)

Replace all corresponding hard-coded literal values in the file with the new constant names.

**ioutil replacements** (spec 3.8.8):
- `ioutil.ReadDir(...)` → `os.ReadDir(...)` in `getTmpFiles`
- `ioutil.ReadAll(resp.Body)` → `io.ReadAll(resp.Body)` in the M3U8 content-type case
- Remove `"io/ioutil"` from imports (keep `"io"`)

**strings.Contains** (spec 3.8.9):
- `strings.Index(stream.URL, ".m3u8") != -1` → `strings.Contains(stream.URL, ".m3u8")`

**`_ = i` loop cleanup** (spec 3.8.10):
- Stream-limit loop: replace `for i := 1; i < 60; i++ { _ = i; ... }` with `for range [streamLimitLoopCount]struct{}{}{ ... }`
- HLS sleep sub-loops: replace `for i := 0.0; i < sleep*1000; i = i+100 { _ = i; ... }` with a deadline-based loop: `deadline := time.Now().Add(time.Duration(sleep * float64(time.Second))); for time.Now().Before(deadline) { time.Sleep(hlsPollingStepMS * time.Millisecond); ... }`
- `parseM3U8` loop: `for i, line := range lines { _ = i; ... }` → `for _, line := range lines { ... }`

**Debug log for ParseFloat error** (spec 3.8.11):
- In `getTmpFiles`: when `strconv.ParseFloat` fails, add `showDebug(fmt.Sprintf("Buffer Status:Skipping non-numeric temp file: %s", file.Name()), 2)` and `continue` instead of silently skipping

**Goroutine entry/exit logging** (spec 3.8.12):
- At entry of `connectToStreamingServer`: `showDebug(fmt.Sprintf("Buffer Goroutine:START ..."), 1)`
- `defer showDebug(fmt.Sprintf("Buffer Goroutine:END ..."), 1)` at top of function
- Same for `thirdPartyBuffer`

Verification: `go build ./...` and `go vet ./...` pass.

### [x] Step: `src/buffer.go` — Part B: Streaming loop fixes (Loop 2 / client-send loop)
<!-- chat-id: e934287b-5315-4802-a30f-ea616b25bcf0 -->

Spec reference: 3.8.2, 3.8.3, 3.8.4 / bugs 1.4, 1.5, 1.1.

**Replace `http.CloseNotifier` with context** (spec 3.8.2 / bug 1.4):
- Remove the `cn, ok := w.(http.CloseNotifier)` type assertion and the `cn.CloseNotify()` channel usage
- Replace with a `select` on `r.Context().Done()` to detect client disconnect; call `killClientConnection` and `return` on cancellation
- Also check `BufferClients` entry for an error on the `default` branch

**Inactivity timeout for Loop 2** (spec 3.8.3 / bug 1.5):
- Before Loop 2: declare `lastSegmentTime := time.Now()`
- At the top of each Loop 2 iteration: check `time.Since(lastSegmentTime) > bufferInactivityTimeoutIterations*100*time.Millisecond`; if exceeded, log, call `killClientConnection`, and `return`
- Also check `BufferClients.Load(playlistID + stream.MD5)` exists; if not, `return` immediately
- Update `lastSegmentTime = time.Now()` whenever `getTmpFiles` returns a non-empty list

**Fix `defer file.Close()` inside inner loop** (spec 3.8.4 / bug 1.1):
- In the `for _, f := range tmpFiles` loop inside Loop 2: replace `defer file.Close()` with explicit `file.Close()` calls immediately after use (after reading bytes from the file, before `continue` or next iteration)

Verification: `go build ./...` and `go vet ./...` pass.

### [x] Step: `src/buffer.go` — Part C: `connectToStreamingServer` and `thirdPartyBuffer` fixes
<!-- chat-id: f3926cae-ac1d-479e-8845-ff7f94636505 -->

Spec reference: 3.8.5, 3.8.6, 3.8.7 / bugs 1.1, 1.2, 1.3.

**Fix `defer resp.Body.Close()` inside outer loop and goto block** (spec 3.8.5 / bug 1.1):
- Audit every `defer resp.Body.Close()` inside `connectToStreamingServer`'s outer `for` loop and the `Redirect` goto block (~lines 707, 716, 727, 733, 737, 880, 891)
- Replace scattered defers with explicit `resp.Body.Close()` calls on every early-return/goto path
- Place a single `defer resp.Body.Close()` only after all redirect handling has resolved and `resp` is known valid, immediately before the `switch contentType` block
- Likewise replace `defer bufferFile.Close()` inside the TS read inner loop with an explicit call after use

**Fix `panic(err)` in `thirdPartyBuffer`** (spec 3.8.6 / bug 1.2):
- On the `os.OpenFile` error path (~line 1513): replace `panic(err)` with `ShowError(err, 0)`, `addErrorToStream(err)`, `cmd.Process.Kill()`, `cmd.Wait()`, `return`

**Fix dual file handle in `thirdPartyBuffer`** (spec 3.8.7 / bug 1.3):
- In the segment rotation loop (~lines 1609–1611): remove the `os.OpenFile` call that follows `os.Create` on the same `tmpFile`; use only `os.Create` (which returns a writable, truncated handle); remove the now-unused `errCreate`/`errOpen` variables

Verification: `go build ./...` and `go vet ./...` pass.

### [x] Step: Replace `ioutil` in remaining source files
<!-- chat-id: d3dc2f8e-6cfa-45da-a9b1-d2cb7e1110c7 -->

Spec reference: 3.9, 3.10 / bug 4.1.

Apply mechanical `ioutil` → `io`/`os` replacements and remove `"io/ioutil"` imports in:

- **`src/webserver.go`**: `ioutil.ReadAll` → `io.ReadAll`, `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile`, `ioutil.ReadDir` → `os.ReadDir`; also replace `strings.Index(...) != -1` → `strings.Contains(...)` (spec 3.9)
- **`src/xepg.go`**: `ioutil.ReadAll` → `io.ReadAll`, `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile`
- **`src/backup.go`**: `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile`
- **`src/provider.go`**: `ioutil.ReadAll` → `io.ReadAll`, `ioutil.ReadFile` → `os.ReadFile`
- **`src/update.go`**: `ioutil.ReadAll` → `io.ReadAll`
- **`src/html-build.go`**: `ioutil.ReadFile` → `os.ReadFile`
- **`src/internal/authentication/authentication.go`**: `ioutil.ReadFile` → `os.ReadFile`, `ioutil.WriteFile` → `os.WriteFile`
- **`src/internal/m3u-parser/m3u-parser_test.go`**: `ioutil.ReadFile` → `os.ReadFile`
- **`src/internal/up2date/client/client.go`**: `ioutil.ReadAll` → `io.ReadAll`
- **`src/internal/imgcache/cache.go`**: all `ioutil` variants → `io`/`os` equivalents

Ensure `"io"` and/or `"os"` are in each file's imports; remove `"io/ioutil"` from all.

Verification: `go build ./...` and `go vet ./...` pass. `go test ./src/internal/m3u-parser/...` passes.

### [x] Step: Expand buffer size options in TypeScript UI
<!-- chat-id: 4ebe82e8-3e5c-4f40-810e-2dd0bf57ba40 -->

Spec reference: 3.11 / requirement 5.2.

**`ts/settings_ts.ts`** (lines ~348–349):
- Change `text` array to: `["0.5 MB", "1 MB", "2 MB", "4 MB", "8 MB", "16 MB", "32 MB", "64 MB", "128 MB"]`
- Change `values` array to: `["512", "1024", "2048", "4096", "8192", "16384", "32768", "65536", "131072"]`

Regenerate `src/webUI.go`:
```bash
cd ts && bash compileJS.sh
```

Verification: `go build ./...` passes (webUI.go is valid Go). Confirm the new size options appear in the compiled output by grepping `webUI.go` for `131072`.

### [x] Step: Final verification
<!-- chat-id: 06788a9e-4038-4275-8eaa-30aff87c739b -->

Run the full verification suite from spec section 6:

```bash
go build ./...
go vet ./...
go test ./src/internal/m3u-parser/...
cd ts && bash compileJS.sh
```

Record results in this plan. All commands must exit 0. If any failures remain, fix them before marking this step complete.
