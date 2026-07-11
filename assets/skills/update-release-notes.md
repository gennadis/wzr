# Skill: update-release-notes

Update a Confluence page with release notes for the given project and version.

## Parameters

- `project` — project name (e.g. `MYAPP`)
- `version` — release version string (e.g. `1.2.3`)
- `space` — Confluence space key where the release notes page lives

## Instructions

1. Use the Jira MCP to fetch all issues for project `{{ .project }}` with fix version `{{ .version }}` that are in status "Done" or "Released".
2. Group issues by type: Bug Fixes, Features, Improvements, Other.
3. Use the Confluence MCP to find the release notes page in space `{{ .space }}` (search for "Release Notes {{ .project }}").
   - If the page exists, append a new section for version `{{ .version }}`.
   - If it does not exist, create a new page titled "Release Notes {{ .project }}".
4. Write the release notes section in this format:

```
## v{{ .version }} — <today's date>

### Features
- PROJ-123: Short issue summary

### Bug Fixes
- PROJ-456: Short issue summary

### Improvements
- PROJ-789: Short issue summary
```

5. Confirm the page was updated and output its URL.

## Expected output

```
Updated Confluence release notes for {{ .project }} v{{ .version }}
Page URL: https://confluence.example.com/...
Issues included: N
```
