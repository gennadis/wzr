# WZR Manual Test Plan

**App URL:** `http://localhost:8080` (default port)  
**Prerequisite:** WZR binary running with Qwen Code CLI available.

---

## TC-01 — Home Page Loads and Stats Bar Renders

**Page:** `/`

**Steps:**
1. Open the app in a browser.
2. Verify the nav bar shows: WZR brand, Home (active), Creator, Dashboard, Templates.
3. Verify the stats bar shows three tiles: Runs, Hours Saved, Success Rate.
4. Verify the three feature cards are visible: Pipeline Creator, Execution Dashboard, Templates Gallery.
5. Click each card and confirm it navigates to the correct page.

**Expected:** Stats bar values load (may show `0` on a fresh install, not `—`). All three cards are clickable links.

---

## TC-02 — Templates Gallery: Browse and Clone

**Page:** `/templates`

**Steps:**
1. Navigate to Templates.
2. Verify the grid shows at least 4 built-in templates (release-manager, incident-response, onboarding-checklist, pr-review-automation).
3. Each card should display: name, description, step count.
4. Click **Clone & Use** on `release-manager`.
5. Verify the status line below the button changes to `Cloned! Go to Dashboard to run it.` in green.
6. Click **Clone & Use** on the same template a second time.
7. Verify the operation still succeeds (overwrite is allowed) without an error.

**Expected:** Templates load, clone succeeds, feedback is immediate. No page reload required.

---

## TC-03 — Creator: Sidebar Loads Skills and MCPs

**Page:** `/creator`

**Steps:**
1. Navigate to Creator.
2. In the left sidebar verify the **Skills** section lists items (e.g., `check-release-readiness`, `update-release-notes`).
3. Verify the **MCP Servers** section lists 5 servers: bitbucket, jira, jenkins, confluence, postgres.
4. Click one of the skill names in the sidebar.
5. Verify the chat input is pre-filled with `Use the "<skill-name>" skill to `.
6. Verify the chat input field gets focus automatically.

**Expected:** Both sidebar sections populated on load. Skill click pre-fills input.

---

## TC-04 — Creator: Step-by-Step Chat Flow

**Page:** `/creator` → Step by Step tab

**Steps:**
1. In the chat input type `Add a Jira search step to find ready tickets` and press Send.
2. Verify a "thinking" animation (three bouncing dots) appears while the request is in flight.
3. Verify the thinking animation disappears when the response arrives.
4. If AI returns a step suggestion: verify a card appears showing the step name, type badge, and JSON preview, plus an **Add to pipeline** button.
5. Click **Add to pipeline**.
6. Verify the step chip appears in the **Accumulated steps** area with the correct name.
7. Verify the step count label updates to `(1)`.
8. Click the `×` on the chip.
9. Verify the chip is removed and the count returns to `(0)`.
10. Send a conversational message like `Hello!`.
11. Verify the AI responds with a plain text message (not a step card).

**Expected:** Chat loop works end-to-end. JSON step and plain-text replies are handled separately.

---

## TC-05 — Creator: Describe & Generate

**Page:** `/creator` → Describe & Generate tab

**Steps:**
1. Switch to the **Describe & Generate** tab.
2. In the textarea enter: `Check Jira tickets are ready, then trigger a Jenkins build`.
3. Click **Generate YAML**.
4. Verify the button area shows `Generating…` while in progress.
5. Verify the YAML preview area populates with syntax-highlighted YAML when done.
6. Verify the YAML contains a `name:` field, `steps:` list, and at least one step of type `mcp` or `skill`.
7. Verify no ` ``` ` markdown fences appear in the output.
8. Verify YAML keys are colored blue and template expressions `{{ .x }}` are colored orange.

**Expected:** YAML generated and displayed with syntax highlighting, no raw fences.

---

## TC-06 — Creator: Save Pipeline (Step-by-Step)

**Page:** `/creator` → Step by Step tab

**Pre-condition:** At least one step accumulated (from TC-04).

**Steps:**
1. In the **Pipeline name** field at the bottom enter `test-manual-pipeline`.
2. Click **Save Pipeline**.
3. Verify the save status shows `Saved!` in green (disappears after ~3 s).
4. Navigate to Dashboard.
5. Verify the pipeline `test-manual-pipeline` appears in the pipeline grid.

**Expected:** Pipeline persisted and visible on Dashboard.

---

## TC-07 — Creator: Save Pipeline (Generated YAML)

**Page:** `/creator` → Describe & Generate tab

**Pre-condition:** YAML generated (from TC-05).

**Steps:**
1. Leave the pipeline name field empty (or optionally type an override name).
2. Click **Save Pipeline**.
3. Verify it saves using the `name:` embedded in the generated YAML.
4. Optionally type a different name in the name field and save again.
5. Verify the pipeline appears on Dashboard under the overridden name.

**Expected:** Generated YAML is saved; name field overrides the YAML `name:` field when provided.

---

## TC-08 — Creator: Save Validation Errors

**Page:** `/creator`

**Steps:**
1. On the **Step by Step** tab with zero steps, click **Save Pipeline**.
2. Verify the save status shows `Add at least one step first.`
3. Add one step, leave the name field empty, click **Save Pipeline**.
4. Verify the save status shows `Enter a pipeline name first.`
5. Switch to **Describe & Generate** with no YAML generated, click **Save Pipeline**.
6. Verify the save status shows `Generate a pipeline first.`

**Expected:** All three validation messages appear correctly; no network request is made.

---

## TC-09 — Dashboard: Pipeline List and Params Modal

**Page:** `/dashboard`

**Pre-condition:** At least one pipeline saved (e.g., `release-manager` cloned in TC-02).

**Steps:**
1. Navigate to Dashboard.
2. Verify the pipeline grid shows each saved pipeline as a card with name, step count, and optional description.
3. Click **Run** on the `release-manager` pipeline.
4. Verify a modal appears titled `Run: release-manager`.
5. Verify the modal contains labeled input fields for each declared param (VERSION, JENKINS_JOB, etc.).
6. Fill in all fields with dummy values (e.g., `1.0`, `my-job`, `my-repo`, `my-pr`, `my-page`, `TEST`, `key=val`).
7. Click **Cancel**.
8. Verify the modal closes and no run starts.

**Expected:** Modal renders all params. Cancel closes without starting a run.

---

## TC-10 — Dashboard: Run Execution and Live SSE Events

**Page:** `/dashboard`

**Pre-condition:** A pipeline with at least one step is saved and Qwen is available.

**Steps:**
1. Click **Run** on a pipeline, fill in params, click **Run** in the modal.
2. Verify the run panel slides into view with the pipeline name and a `run_id` label.
3. Verify the step list shows each step with a `Pending` badge initially.
4. As the run progresses verify each active step badge changes: `Pending` → `Running` → `Done`.
5. Verify the **Live Narration** sidebar receives lines of text as the run proceeds.
6. Narration auto-scrolls to the bottom — verify by making the log tall and watching it stay at the bottom.
7. After all steps complete verify the green **success banner** appears.
8. Verify the **Post-run Chat** panel becomes visible after completion.

**Expected:** Step badges update in real time via SSE. Narration streams live. Banner appears on finish.

---

## TC-11 — Dashboard: Approval Gate

**Pre-condition:** A pipeline that contains an `approval` step type is running.

**Steps:**
1. Start a run that includes a human-approval step.
2. Wait for the approval step to reach `Waiting` (yellow badge).
3. Verify **Approve** and **Reject** buttons appear on that step row.
4. Click **Approve**.
5. Verify the buttons disappear and the step proceeds (badge changes to `Running` then `Done`).
6. Start another run with the same pipeline.
7. At the approval gate click **Reject**.
8. Verify the pipeline transitions to `failed` state and the red failed banner appears.

**Expected:** Approve continues the run; Reject fails it.

---

## TC-12 — Dashboard: Self-Healing Repair Flow

**Pre-condition:** A pipeline step can be configured to fail so the repair flow triggers. (Check that runner emits `repair_suggested` on step failure.)

**Steps:**
1. Start a run where a step is expected to fail (e.g., wrong MCP params or a skill that errors).
2. When the step fails verify the badge changes to `Repair` (orange).
3. Verify a **repair card** appears under the step showing: diagnosis text and a fix command in monospace.
4. Click **Apply & Retry**.
5. Verify the buttons disappear and the runner attempts the step again (badge goes back to `Running`).
6. Alternatively start another failing run and click **Give Up** on the repair card.
7. Verify the pipeline moves to `postmortem` state.
8. Verify a `Postmortem` badge (purple) appears and a scrollable postmortem text panel is shown.
9. Verify the failed banner reads `Pipeline failed. See post-mortem below.`
10. Verify the Post-run Chat panel becomes visible after postmortem.

**Expected:** Repair card appears with correct content. Both paths (retry / give up) work correctly.

---

## TC-13 — Dashboard: Post-Run Chat

**Pre-condition:** A run has finished (success or failure), Post-run Chat panel is visible.

**Steps:**
1. Type a question in the chat input: `Which steps succeeded?`
2. Press Enter (or click **Ask**).
3. Verify a user bubble appears with the question text.
4. Verify an AI bubble appears with a concise answer from Qwen.
5. Ask a follow-up question: `How long did the Jenkins step take?`
6. Verify the conversation grows with a second Q&A pair.
7. Verify the chat bubbles area auto-scrolls to the latest message.

**Expected:** Q&A works without page reload. Both user and AI bubbles render correctly.

---

## TC-14 — Dashboard: Stats Bar

**Page:** `/dashboard`

**Steps:**
1. Note the current **Runs** count before running anything.
2. Complete a successful pipeline run (TC-10).
3. Click **Refresh** in the stats bar.
4. Verify the **Runs** count has incremented.
5. Verify **Hours Saved** shows a non-zero value (based on `manual_minutes` from the pipeline).
6. Verify **Success Rate** shows a percentage between 0% and 100%.

**Expected:** Stats reflect completed runs accurately after refresh.

---

## TC-15 — Navigation: All Links and Active State

**Steps:**
1. From each page (Home, Creator, Dashboard, Templates) click every nav link and verify the correct page loads.
2. On each page verify exactly one nav link has the `active` styling.
3. Verify the WZR brand link in the nav always navigates to Home.

**Expected:** Nav links work from all pages. Active highlight matches current page.

---

## TC-16 — Edge Case: No Pipelines Saved

**Page:** `/dashboard`

**Pre-condition:** All pipelines deleted from `pipelines/` directory (or fresh install with none cloned).

**Steps:**
1. Open Dashboard.
2. Verify the pipeline grid shows the message `No pipelines saved yet.` with a link to Creator.
3. Click the Creator link in that message.
4. Verify it navigates to `/creator`.

**Expected:** Empty-state message is shown; link works.

---

## TC-17 — Edge Case: Run on Pipeline Without Params

**Pre-condition:** A saved pipeline with no `params` block (or empty params).

**Steps:**
1. Click **Run** on that pipeline.
2. Verify the modal opens and shows `No parameters required.` instead of input fields.
3. Click **Run**.
4. Verify the run starts normally.

**Expected:** No param inputs shown for param-less pipelines. Run starts cleanly.
