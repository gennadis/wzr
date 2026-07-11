# Browser Test Execution — WZR Manual QA

## Overview
Execute all 17 manual test cases from `docs/manual-test-plan.md` using Chrome browser automation.
The goal is to validate the full WZR UI end-to-end: navigation, Pipeline Creator, Templates Gallery,
Dashboard run lifecycle (SSE events, approval gates, repair/postmortem), and post-run chat.

Each task maps directly to one or more test cases from the source document. Tasks run in a fixed
order so preconditions are met (empty-state check before any cloning, approval/repair tests last).

## Context (from discovery)
- Files involved: `docs/manual-test-plan.md` (17 TCs, source of truth), `assets/web/static/*.html`
- Pages under test: `/` (Home), `/creator`, `/dashboard`, `/templates`
- Backend: Go binary on `http://localhost:8080`; Qwen Code CLI used for TC-04, TC-05, TC-10–13
- Existing saved pipelines: `pipelines/0001.yaml`, `incident-response.yaml`, `onboarding-checklist.yaml`, `release-manager.yaml`
- Approval-gate pipeline: `0001.yaml` or `release-manager.yaml` (contain `type: approval` steps)

## Development Approach
- **Testing approach**: live browser — run against the real server; fail fast if Qwen is unavailable
- Execute tasks in order — later tasks depend on state created by earlier ones
- Mark each checkbox `[x]` immediately when verified; if a check fails, note with ⚠️ and continue
- TC-16 (empty-state) MUST run before any template is cloned or pipeline is saved

## Testing Strategy
- Each task = one or two TCs from `docs/manual-test-plan.md`
- Browser automation via `mcp__claude-in-chrome__*` tools: navigate, read_page, computer (click/type), gif_creator for key flows
- AI-dependent tests (TC-04, TC-05, TC-10–13): run live; if Qwen binary is missing, mark task ⚠️ BLOCKED and skip
- Record GIFs for: TC-10 (run lifecycle), TC-11 (approval gate), TC-12 (repair/postmortem)

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered issues with ⚠️ prefix
- If a check fails, record the failure detail inline and continue to the next item

## Solution Overview
Run the 17 TCs sequentially via browser automation. Each task opens the relevant URL, performs
the actions described in the TC, asserts the expected outcomes by reading page content or
inspecting DOM state, then marks the checklist. At the end produce a pass/fail summary table.

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): each browser test execution task
- **Post-Completion**: final results summary, any bug report links

---

## Implementation Steps

### Task 1: Preflight — server reachable, browser ready

**Files:** (no source changes — browser automation only)

- [ ] open a new Chrome tab via `tabs_create_mcp`
- [ ] navigate to `http://localhost:8080`
- [ ] verify HTTP 200 and the WZR page title is present (read_page)
- [ ] if server is unreachable: mark all subsequent tasks ⚠️ BLOCKED and stop

### Task 2: TC-01 — Home page loads and stats bar renders

**Files:** (no source changes)

- [ ] verify nav bar contains: WZR brand, Home (active), Creator, Dashboard, Templates
- [ ] verify stats bar shows three tiles: Runs, Hours Saved, Success Rate
- [ ] verify tile values are numeric (not `—`) after page settles
- [ ] verify three feature cards are visible: Pipeline Creator, Execution Dashboard, Templates Gallery
- [ ] click each card and confirm navigation to the correct URL (`/creator`, `/dashboard`, `/templates`)
- [ ] navigate back to `/` after each check

### Task 3: TC-15 — Navigation links and active state

**Files:** (no source changes)

- [ ] from `/` verify Home nav link has active styling; Creator, Dashboard, Templates do not
- [ ] navigate to `/creator`; verify Creator link is active, others are not
- [ ] navigate to `/dashboard`; verify Dashboard link is active
- [ ] navigate to `/templates`; verify Templates link is active
- [ ] click the WZR brand link from `/templates`; verify it returns to `/`

### Task 4: TC-16 — Empty state when no pipelines exist

**Files:** (no source changes)

- [ ] navigate to `/dashboard`
- [ ] if pipelines already exist in the grid, note ⚠️ precondition not met (skip empty-state check)
- [ ] if grid shows empty state: verify message `No pipelines saved yet.` with a Creator link
- [ ] click the Creator link in the empty-state message and verify it navigates to `/creator`

### Task 5: TC-02 — Templates gallery: browse and clone

**Files:** (no source changes)

- [ ] navigate to `/templates`
- [ ] verify grid contains at least 4 template cards
- [ ] verify each card shows: name, description, step count
- [ ] click **Clone & Use** on `release-manager`
- [ ] verify status text changes to `Cloned! Go to Dashboard to run it.` in green
- [ ] click **Clone & Use** on `release-manager` a second time
- [ ] verify second clone succeeds (no error shown)

### Task 6: TC-03 — Creator sidebar loads skills and MCPs

**Files:** (no source changes)

- [ ] navigate to `/creator`
- [ ] verify **Skills** sidebar section lists at least one skill name
- [ ] verify **MCP Servers** section lists 5 servers: bitbucket, jira, jenkins, confluence, postgres
- [ ] click one skill name in the sidebar
- [ ] verify the chat input is pre-filled with `Use the "<skill-name>" skill to `
- [ ] verify the chat input field has focus (cursor visible / no extra click needed)

### Task 7: TC-08 — Creator save validation errors

**Files:** (no source changes)

- [ ] on **Step by Step** tab with zero steps, click **Save Pipeline**
- [ ] verify status shows `Add at least one step first.`
- [ ] send one chat message so a step is added, then clear the pipeline-name field, click **Save Pipeline**
- [ ] verify status shows `Enter a pipeline name first.`
- [ ] switch to **Describe & Generate** tab without generating YAML, click **Save Pipeline**
- [ ] verify status shows `Generate a pipeline first.`

### Task 8: TC-04 — Creator step-by-step chat flow

**Files:** (no source changes)

- [ ] navigate to `/creator`, **Step by Step** tab
- [ ] type `Add a Jira search step to find ready tickets` and click Send
- [ ] verify thinking animation (three dots) appears while request is in flight
- [ ] verify thinking animation disappears when response arrives
- [ ] if AI returns a step card: verify it shows name, type badge, JSON preview, and **Add to pipeline** button
- [ ] click **Add to pipeline**; verify step chip appears in Accumulated steps area with count `(1)`
- [ ] click `×` on the chip; verify chip removed and count returns to `(0)`
- [ ] send `Hello!`; verify AI responds with a plain text message (no step card shown)
- [ ] if Qwen unavailable: mark ⚠️ BLOCKED, skip to Task 9

### Task 9: TC-05 — Creator describe & generate

**Files:** (no source changes)

- [ ] switch to **Describe & Generate** tab
- [ ] enter `Check Jira tickets are ready, then trigger a Jenkins build` in the textarea
- [ ] click **Generate YAML**
- [ ] verify status area shows `Generating…` while in progress
- [ ] verify YAML preview populates when done (non-empty, no ` ``` ` fences)
- [ ] verify YAML contains `name:` field and `steps:` list with at least one step
- [ ] verify YAML keys are colored blue and `{{ .x }}` expressions are colored orange
- [ ] if Qwen unavailable: mark ⚠️ BLOCKED

### Task 10: TC-06 — Save pipeline from step-by-step

**Files:** (no source changes)

- [ ] on **Step by Step** tab, ensure at least one step is accumulated (from Task 8 or add a new one)
- [ ] enter `test-manual-pipeline` in the pipeline-name field
- [ ] click **Save Pipeline**
- [ ] verify status shows `Saved!` in green, then disappears after ~3 s
- [ ] navigate to `/dashboard` and verify `test-manual-pipeline` appears in the pipeline grid

### Task 11: TC-07 — Save pipeline from generated YAML

**Files:** (no source changes)

- [ ] on **Describe & Generate** tab with YAML already generated (from Task 9)
- [ ] leave the pipeline-name field empty and click **Save Pipeline**
- [ ] verify it saves using the name embedded in the YAML; confirm on Dashboard
- [ ] return to Creator, type an override name (e.g., `overridden-pipeline`), click **Save Pipeline**
- [ ] verify pipeline appears on Dashboard under the overridden name

### Task 12: TC-09 — Dashboard pipeline list and params modal

**Files:** (no source changes)

- [ ] navigate to `/dashboard`
- [ ] verify pipeline grid shows at least one card with name, step count, and optional description
- [ ] click **Run** on `release-manager`
- [ ] verify params modal opens titled `Run: release-manager`
- [ ] verify modal contains labeled input fields for each declared param (VERSION, JENKINS_JOB, etc.)
- [ ] fill all fields with dummy values (e.g., `1.0`, `my-job`, `my-repo`, `PR-1`, `my-page`, `TEST`, `key=val`)
- [ ] click **Cancel**; verify modal closes and no run panel appears

### Task 13: TC-17 — Run modal when pipeline has no params

**Files:** (no source changes)

- [ ] identify or create a pipeline with no `params` block (e.g., `test-manual-pipeline` from Task 10 if it has no params)
- [ ] click **Run** on that pipeline
- [ ] verify modal shows `No parameters required.` instead of input fields
- [ ] click **Run** in the modal; verify the run panel appears and a run starts

### Task 14: TC-10 — Run execution and live SSE events

**Files:** (no source changes)

- [ ] start a GIF recording (`gif_creator`)
- [ ] click **Run** on a pipeline with multiple steps, fill params, click **Run**
- [ ] verify run panel slides into view with pipeline name and `run_id` label
- [ ] verify each step row appears with `Pending` badge initially
- [ ] verify step badges update in sequence: `Pending` → `Running` → `Done`
- [ ] verify Live Narration sidebar receives streaming text lines
- [ ] verify narration log auto-scrolls to the bottom as lines arrive
- [ ] after all steps complete, verify green success banner appears
- [ ] verify Post-run Chat panel becomes visible after completion
- [ ] stop and save GIF as `wzr-run-lifecycle.gif`
- [ ] if Qwen unavailable: mark ⚠️ BLOCKED

### Task 15: TC-11 — Approval gate (approve and reject paths)

**Files:** (no source changes)

- [ ] start a GIF recording
- [ ] run `release-manager` (or `0001.yaml`) pipeline — it contains a `type: approval` step
- [ ] wait for approval step to show `Waiting` badge (yellow)
- [ ] verify **Approve** and **Reject** buttons appear on that step row
- [ ] click **Approve**; verify buttons disappear and step proceeds (badge → `Running` → `Done`)
- [ ] verify the pipeline eventually completes (success banner)
- [ ] run the same pipeline again; at approval gate click **Reject**
- [ ] verify pipeline transitions to failed state and red failed banner appears
- [ ] stop and save GIF as `wzr-approval-gate.gif`

### Task 16: TC-12 — Self-healing repair flow and postmortem

**Files:** (no source changes)

- [ ] start a GIF recording
- [ ] run a pipeline with a step that will fail (wrong params or broken MCP tool call)
- [ ] when step fails, verify badge changes to `Repair` (orange)
- [ ] verify repair card appears showing diagnosis text and fix command in monospace
- [ ] click **Apply & Retry**; verify buttons disappear and step retries (`Running` badge)
- [ ] run the pipeline again; let same step fail; on repair card click **Give Up**
- [ ] verify badge changes to `Postmortem` (purple)
- [ ] verify postmortem text panel appears with scrollable content
- [ ] verify failed banner reads `Pipeline failed. See post-mortem below.`
- [ ] verify Post-run Chat panel becomes visible after postmortem
- [ ] stop and save GIF as `wzr-repair-postmortem.gif`
- [ ] if repair flow cannot be triggered: mark ⚠️ BLOCKED and note which step/pipeline to use

### Task 17: TC-13 — Post-run chat

**Files:** (no source changes)

- [ ] after a completed run (success or failure), verify Post-run Chat panel is visible
- [ ] type `Which steps succeeded?` and press Enter
- [ ] verify a user bubble appears with the question text
- [ ] verify an AI bubble appears with a concise answer
- [ ] type `How long did the Jenkins step take?` and press Enter
- [ ] verify a second Q&A pair appears in the chat
- [ ] verify chat bubbles area auto-scrolls to the latest message
- [ ] if Qwen unavailable: mark ⚠️ BLOCKED

### Task 18: TC-14 — Stats bar refresh after a run

**Files:** (no source changes)

- [ ] navigate to `/dashboard` and note current **Runs** count before running
- [ ] complete a successful pipeline run (or use the one from Task 14 if still in session)
- [ ] click **Refresh** in the stats bar
- [ ] verify **Runs** count has incremented by at least 1
- [ ] verify **Hours Saved** shows a non-zero value
- [ ] verify **Success Rate** shows a percentage between 0% and 100%

### Task 19: Verify acceptance criteria

- [ ] confirm all 17 TCs have been attempted (tasks 2–18 above)
- [ ] list any tasks marked ⚠️ BLOCKED or ⚠️ FAILED with brief reason
- [ ] verify no TC was silently skipped without a recorded reason
- [ ] stop any active GIF recordings

### Task 20: Results summary

- [ ] produce a pass/fail table covering all 17 TCs (see Post-Completion below)
- [ ] move this plan to `docs/plans/completed/` (`mkdir -p docs/plans/completed`)

---

## Post-Completion

**Results table** (fill in after execution):

| TC | Name | Result | Notes |
|----|------|--------|-------|
| TC-01 | Home page | — | |
| TC-02 | Templates clone | — | |
| TC-03 | Creator sidebar | — | |
| TC-04 | Step-by-step chat | — | |
| TC-05 | Describe & Generate | — | |
| TC-06 | Save from step-by-step | — | |
| TC-07 | Save from generated YAML | — | |
| TC-08 | Save validation errors | — | |
| TC-09 | Params modal | — | |
| TC-10 | Run execution + SSE | — | |
| TC-11 | Approval gate | — | |
| TC-12 | Repair + postmortem | — | |
| TC-13 | Post-run chat | — | |
| TC-14 | Stats bar | — | |
| TC-15 | Navigation | — | |
| TC-16 | Empty state | — | |
| TC-17 | No-params modal | — | |

**GIF artifacts** (saved during execution):
- `wzr-run-lifecycle.gif` — Task 14
- `wzr-approval-gate.gif` — Task 15
- `wzr-repair-postmortem.gif` — Task 16

**Bug reports**: file issues for any ⚠️ FAILED items before marking the plan complete.
