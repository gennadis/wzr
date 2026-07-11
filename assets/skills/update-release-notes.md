# Skill: update-release-notes

Update a Confluence page with release notes for the given project and version.

## Parameters

- `project` — Jira project key (e.g. `MYAPP`)
- `version` — release version string (e.g. `1.2.3`)
- `space` — Confluence space key where the release notes page lives (e.g. `TEAM`)
- `build_url` — Jenkins build URL for traceability (optional)

## Instructions

1. Use the Jira MCP `search_issues` tool to fetch all completed issues:
   ```
   project = {{ .project }} AND fixVersion = "{{ .version }}" AND status in (Done, Released, Closed)
   ORDER BY issuetype ASC, key ASC
   ```
2. Group issues by type: `Story`/`Feature` → Features, `Bug` → Bug Fixes, `Task`/`Improvement` → Improvements, everything else → Other.
3. Use the Confluence MCP `get_page` to find a page titled "Release Notes {{ .project }}" in space `{{ .space }}`.
   - If found: use `update_page` to append the new version section.
   - If not found: use `create_page` to create it from scratch.
4. Format the new section as:

```
## v{{ .version }} — <today's date>

### Features
- PROJ-123: Short issue summary (assignee)

### Bug Fixes
- PROJ-456: Short issue summary (assignee)

### Improvements
- PROJ-789: Short issue summary (assignee)

Build: {{ .build_url }}
```

5. After updating, output the Confluence page URL and a count of issues included.
6. If no issues are found, still create the version section with a note: "No issues linked to this version."

## Expected output

```
Updated Confluence release notes for {{ .project }} v{{ .version }}
Page URL: https://confluence.sber.ru/display/{{ .space }}/Release+Notes+{{ .project }}
Issues included: N (3 features, 2 bug fixes, 1 improvement)
```
