# Auto

## Configuration
- **Artifacts Path**: {@artifacts_path} → `.zenflow/tasks/{task_id}`

## Agent Instructions

Ask the user questions when anything is unclear or needs their input. This includes:
- Ambiguous or incomplete requirements
- Technical decisions that affect architecture or user experience
- Trade-offs that require business context

Do not make assumptions on important decisions — get clarification first.

**Debug requests, questions, and investigations:** answer or investigate first. Do not create a plan upfront — the user needs an answer, not a plan. A plan may become relevant later once the investigation reveals what needs to change.

**For all other tasks**, before writing any code, assess the scope of the actual change (not the prompt length — a one-sentence prompt can describe a large feature). Scale your approach:

- **Trivial** (typo, config tweak, single obvious change): implement directly, no plan needed.
- **Small** (a few files, clear what to do): write 2–3 sentences in `plan.md` describing what and why, then implement. No substeps.
- **Medium** (multiple components, design decisions, edge cases): write a plan in `plan.md` with requirements, affected files, key decisions, verification. Break into 3–5 steps.
- **Large** (new feature, cross-cutting, unclear scope): gather requirements and write a technical spec first (`requirements.md`, `spec.md` in `{@artifacts_path}/`). Then write `plan.md` with concrete steps referencing the spec.

**Skip planning and implement directly when** the task is trivial, or the user explicitly asks to "just do it" / gives a clear direct instruction.

To reflect the actual purpose of the first step, you can rename it to something more relevant (e.g., Planning, Investigation). Do NOT remove meta information like comments for any step.

Rule of thumb for step size: each step = a coherent unit of work (component, endpoint, test suite). Not too granular (single function), not too broad (entire feature). Unit tests are part of each step, not separate.

Update `{@artifacts_path}/plan.md` if it makes sense to have a plan and task has more than 1 big step.

## Plan for Fixing Plex Restreaming

The issue is that when streaming in restreaming/buffering mode, `./src/buffer.go` sets `Content-Length: 0` on HTTP stream responses. This causes strict HTTP clients like Plex and FFmpeg to expect 0 bytes and immediately close or ignore the stream. To fix this, we will remove the `Content-Length` header manipulation in `./src/buffer.go` so that the streaming client receives chunked or unconstrained data correctly. We will also improve content-type detection by falling back to `video/mp2t` when `application/octet-stream` is returned, ensuring Plex correctly parses the stream format.

## Fix: Session Expiration Bug (Cookie Path Mismatch)

**Root cause**: JavaScript set the `Token` cookie without `; path=/`, causing a duplicate cookie at path `/web/` while the server set it at path `/`. The `getCookie()` function returned `undefined` when two cookies with the same name existed, breaking token auth on the periodic `updateLog` call (every 10 seconds).

**Files changed**:
- `./ts/network_ts.ts` - Added `; path=/` to cookie set, fixed `getCookie` to handle `>= 2` parts
- `./html/js/network_ts.js` - Same fixes in compiled JS
- `./html/js/data.js` - Added `; path=/` to cookie set in `updateXteveStatus`
- `./src/webUI.go` - Updated base64-embedded JS files

### [x] Step: Fix Plex Restreaming Issue
### [x] Step: Fix Session Expiration Bug
### [x] Step: Verify Build and Tests
### [x] Step: Commit and Release

