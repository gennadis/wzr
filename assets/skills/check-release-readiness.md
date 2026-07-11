# Skill: check-release-readiness

Check whether all Jira tickets for the given project and version are ready to release.

## Parameters

- `project` — Jira project key (e.g. `MYAPP`)
- `version` — release version string (e.g. `1.2.3`)

## Instructions

1. Use the Jira MCP to search for all issues in project `{{ .project }}` that are tagged with fix version `{{ .version }}`.
2. For each issue, check its status.
3. Report:
   - Total number of issues found
   - How many are in status "Ready for Release" or "Done"
   - List any issues that are NOT in a release-ready status (provide issue key + current status)
4. If any issues are not release-ready, output a clear FAIL summary listing the blocking tickets.
5. If all issues are release-ready (or no issues found), output a PASS summary.

## Expected output

```
Release readiness check for {{ .project }} v{{ .version }}
Total issues: N
Ready: N
Blocking: N

PASS / FAIL
Blocking tickets (if any):
  - PROJ-123: In Progress
  - PROJ-456: In Review
```
