# Skill: check-release-readiness

Check whether all Jira tickets for the given project and version are ready to release.

## Parameters

- `project` — Jira project key (e.g. `MYAPP`)
- `version` — release version string (e.g. `1.2.3`)

## Instructions

1. Use the Jira MCP `search_issues` tool with the following JQL query:
   ```
   project = {{ .project }} AND fixVersion = "{{ .version }}" ORDER BY status ASC
   ```
2. For each issue returned, record its key and status.
3. Release-ready statuses are: `Done`, `Released`, `Closed`, `Ready for Release`.
4. Build the report:
   - Count total issues found.
   - Count issues in release-ready statuses.
   - List all issues NOT in a release-ready status with their key, summary, and current status.
5. Output a clear verdict:
   - **PASS** — all issues are release-ready (or no issues found for this version).
   - **FAIL** — one or more issues are not release-ready. List all blocking tickets.
6. If FAIL, recommend specific actions: which tickets need to be moved to "Done" or "Ready for Release" before releasing.

## Expected output

```
Release readiness check for {{ .project }} v{{ .version }}
Total issues: N
Ready: N / N
Blocking: N

Verdict: PASS

--- or ---

Verdict: FAIL
Blocking tickets:
  PROJ-123 (Bug): "Login fails on mobile" — In Progress
  PROJ-456 (Story): "User profile redesign" — In Review

Recommended actions:
  - Move PROJ-123 to Done or reassign to next sprint
  - Move PROJ-456 to Done if code review passed
```
