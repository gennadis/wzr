# Browser Test Bugfixes

## Overview
Fix 4 bugs discovered during the browser test execution session (see `docs/plans/completed/20260711-browser-test-execution.md`).
The bugs affect the Home page stats display, the Pipeline Creator's step-suggestion UI, and the Dashboard's
real-time run panel. Together they prevent stats from loading on the Home page, break the "Add to pipeline"
button for steps with single-quoted params, corrupt Qwen YAML output that includes prose before the code fence,
and cause all SSE events to be lost when the client subscribes slightly after the run starts.

## Context (from discovery)
- Files involved:
  - `assets/web/static/index.html` — Bug 1: HTMX `innerHTML` swap dumps raw JSON
  - `assets/web/static/creator.html` — Bug 2: `onclick` single-quote truncation; Bug 3: `stripCodeFences` regex
  - `assets/web/static/dashboard.html` — Bug 4 (client side): `subscribeToEvents` called after events fire; placeholder div missing `.placeholder` class
  - `internal/notify/sse.go` — Bug 4 (server side): Hub has no replay buffer; events emitted before subscribe are lost
  - `internal/notify/notifier_test.go` — existing tests for Hub; will be extended
- Related patterns: dashboard.html already fetches stats via JS (`loadStats()`); index.html should use the same pattern
- Dependencies: no new packages needed; all changes are within existing files

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- JS-only tasks (Bugs 1–3) have no Go tests; verify by reading the updated file and running `go build ./...`
- Bug 4 (SSE hub) must be covered by updated Go unit tests in `notifier_test.go`
- Run `go build ./... && go test ./... && go vet ./...` after every Go change
- No backwards-incompatible API changes

## Testing Strategy
- **Unit tests**: required for Bug 4 (Go). Add `TestHub_ReplayOnSubscribe` and `TestHub_ReplayClearedOnUnsubscribe`.
- **Browser smoke test** (manual, post-completion): reload Home page → stats tiles show numbers not JSON; open Creator → send a step with single-quoted JQL param → Add to pipeline works; generate YAML with prose prefix → fences stripped; run a pipeline → step badges and narration appear live.
- No existing automated browser/e2e suite in the project.

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix

## Solution Overview
- **Bug 1**: Replace `hx-get`/`hx-trigger`/`hx-swap` attributes on `#stats-bar` in `index.html` with a small inline `<script>` that calls `fetch('/api/stats')` on `DOMContentLoaded` and populates the three `.stat-value` spans — mirrors the `loadStats()` pattern already used in `dashboard.html`. Remove the `<script src="htmx.min.js">` tag since it will no longer be needed.
- **Bug 2**: In `appendStepSuggestion()` in `creator.html`, store the step object as `JSON.stringify(step)` in a `data-step` attribute on the button, then use a plain `onclick="addStep(this)"` handler that reads `JSON.parse(this.dataset.step)` — no string interpolation in the attribute value.
- **Bug 3**: Replace the two anchored regexes in `stripCodeFences()` with logic that finds the first ` ``` ` fence occurrence anywhere in the string and extracts content between it and the last closing ` ``` `.
- **Bug 4**: Add a `replay map[string][]StepEvent` field to `Hub`. In `send()`, append every event to the replay slice. In `Subscribe()`, drain the existing replay slice into the new channel before returning it. In `Unsubscribe()`, delete the replay slice. Also fix the "Starting…" placeholder div in `dashboard.html` to have class `placeholder` so `ensureStepRow()` removes it when the first real step row arrives.

## Technical Details
- **`Hub.replay`**: `map[string][]StepEvent`, protected by the existing `Hub.mu` mutex.
- **`Subscribe` replay**: copy slice elements into the buffered channel (capacity 64) before returning — safe because replay events are small and the channel has capacity.
- **Replay growth**: the replay slice grows for the life of a run then is cleared on `Unsubscribe`. Runs are short; memory impact is negligible.
- **`stripCodeFences` logic**: `idx := strings.Index(s, "` + "```" + `")` → if ≥ 0, take `s[idx:]` then trim the opening fence line and trailing fence.

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): all four bugs are in-tree changes
- **Post-Completion**: manual browser smoke test to verify all four fixes visually

---

## Implementation Steps

### Task 1: Fix Home page stats bar (Bug 1 — HTMX innerHTML)

**Files:**
- Modify: `assets/web/static/index.html`

- [ ] remove `hx-get`, `hx-trigger`, `hx-swap` attributes from `<div id="stats-bar">` in `index.html`
- [ ] add `id` attributes to the three `.stat-value` divs: `id="stat-runs"`, `id="stat-hours"`, `id="stat-rate"`
- [ ] add an inline `<script>` block (before `</body>`) that fetches `/api/stats` on `DOMContentLoaded` and populates the three stat value elements; format hours as `X.Xh`, rate as `X%`
- [ ] remove `<script src="/static/htmx.min.js"></script>` from `index.html` (no longer used after this fix)
- [ ] run `go build ./...` to confirm the embedded asset compiles cleanly
- [ ] verify by reading the file that no `hx-` attributes remain and the script is present

### Task 2: Fix Creator "Add to pipeline" onclick (Bug 2 — single-quote truncation)

**Files:**
- Modify: `assets/web/static/creator.html`

- [ ] in `appendStepSuggestion()` change the button's `onclick` attribute: store the serialised step in a `data-step` attribute using a template literal with escaped HTML, e.g. `data-step="${JSON.stringify(step).replace(/"/g, '&quot;')}"`
- [ ] change the button's `onclick` to `onclick="addStep(this)"` (no interpolation)
- [ ] update `addStep(stepOrBtn, el)` to detect when it receives an HTMLElement (the button): read `JSON.parse(btn.dataset.step)` and derive `el = btn.closest('.step-suggestion')`; keep the existing `addStep(step, el)` object call path for backward compat
- [ ] run `go build ./...` to confirm the embedded asset compiles cleanly
- [ ] verify the fix handles JQL values containing single quotes (e.g. `status = 'Ready'`) by inspecting the generated HTML

### Task 3: Fix stripCodeFences prose prefix (Bug 3 — anchored regex)

**Files:**
- Modify: `assets/web/static/creator.html`

- [ ] rewrite `stripCodeFences(text)` to find the index of the first ` ``` ` occurrence using `indexOf`
- [ ] if a fence is found: slice the string from that index, strip the opening fence line (` ```yaml` or ` ``` `), then strip the trailing ` ``` ` (and anything after it)
- [ ] if no fence is found: return `text.trim()` unchanged (Qwen correctly returned bare YAML)
- [ ] run `go build ./...` to confirm the embedded asset compiles cleanly
- [ ] verify the function handles: (a) bare YAML — no change, (b) ` ```yaml\n...\n``` ` at start — fences stripped, (c) prose + ` ```yaml\n...\n``` ` — prose removed, fences stripped

### Task 4: Add SSE Hub replay buffer + fix placeholder class (Bug 4 — SSE race)

**Files:**
- Modify: `internal/notify/sse.go`
- Modify: `internal/notify/notifier_test.go`
- Modify: `assets/web/static/dashboard.html`

- [ ] add `replay map[string][]StepEvent` field to `Hub` struct in `sse.go`; initialise it in `NewHub()`
- [ ] in `Hub.send()`, after writing to the subscriber channel, append the event to `h.replay[runID]` (under the same mutex lock)
- [ ] in `Hub.Subscribe()`, after creating the new channel, replay all buffered events for `runID` into the channel before appending the subscriber and releasing the lock; ensure the channel has capacity for replay events (current buffer is 64 — sufficient for any single run)
- [ ] in `Hub.Unsubscribe()`, delete `h.replay[runID]` after closing the channel to free memory
- [ ] add `TestHub_ReplayOnSubscribe`: publish 3 events before subscribing, then subscribe and verify all 3 are received in order
- [ ] add `TestHub_ReplayClearedOnUnsubscribe`: publish events, unsubscribe, subscribe again — verify no events are replayed on the second subscription
- [ ] run `go test ./internal/notify/... -v` — all tests must pass
- [ ] in `dashboard.html` line 274: add class `placeholder` to the "Starting…" div so `ensureStepRow()` can remove it when the first real step row arrives
- [ ] run `go build ./...` and `go test ./...` — must pass before Task 5

### Task 5: Verify acceptance criteria

- [ ] run full quality gate: `go mod tidy && go mod verify && go build ./... && go test ./... && go vet ./... && golangci-lint run ./...`
- [ ] confirm Bug 1 fix: `index.html` has no `hx-get`/`hx-swap` and includes a `fetch('/api/stats')` script
- [ ] confirm Bug 2 fix: `creator.html` button uses `data-step` attribute and `onclick="addStep(this)"` — no single-quote interpolation
- [ ] confirm Bug 3 fix: `stripCodeFences` uses `indexOf` not anchored regex; handles prose prefix
- [ ] confirm Bug 4 fix: `Hub.replay` map present, populated in `send()`, drained in `Subscribe()`, cleared in `Unsubscribe()`; placeholder div has class `placeholder`
- [ ] confirm all new tests (`TestHub_ReplayOnSubscribe`, `TestHub_ReplayClearedOnUnsubscribe`) pass

### Task 6: Commit and close plan

- [ ] commit all changes with message: `fix: stats bar JSON, creator onclick, stripCodeFences, SSE replay buffer`
- [ ] move this plan to `docs/plans/completed/`

---

## Post-Completion

**Manual browser smoke test** (verify visually after server restart):
1. Open `http://localhost:8080/` — stats bar shows numeric values (not raw JSON)
2. Open Creator → type a step with a param like `status = 'Ready for Development'` → step card appears → click **Add to pipeline** → chip added (no JS error)
3. Open Creator → Describe & Generate → generate YAML → if Qwen prefixes with prose, YAML preview shows clean YAML (no fences, no prose)
4. Open Dashboard → Run any pipeline → step rows appear with live badge updates (Pending → Running → Done) and narration log receives lines
