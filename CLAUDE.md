# CLAUDE.md — WZR Developer Guide

## Quality gate (run before every commit)

```sh
go mod tidy && go mod verify
go build ./...
go test ./...
go vet ./...
golangci-lint run ./...
```

All six must pass. Lint config is in `.golangci.yml`.

## Project structure

```
wzr/
├── assets/           ← package assets; ALL go:embed declarations live here
│   ├── assets.go     (SkillsFS, TemplatesFS, WebStaticFS)
│   ├── skills/       *.md  — skill prompts for Qwen
│   ├── templates/    *.yaml — pipeline templates (embedded at build time)
│   └── web/static/   *.html, *.css, htmx.min.js
├── cmd/wzr/main.go   ← entry point; wires all deps into web.Deps
├── internal/
│   ├── config/       Config struct (Port, QwenBinary, PipelinesDir, HistoryFile)
│   ├── pipeline/     schema.go, parser.go, store.go — YAML definition and persistence
│   ├── runner/       runner.go, state.go, roi.go, runstore.go — execution engine
│   ├── notify/       notifier.go, sse.go, sberchat.go, multi.go — event fan-out
│   ├── qwen/         subprocess.go — Qwen Code CLI wrapper + prompt builder
│   ├── skills/       registry.go — accepts fs.FS; reads *.md files
│   ├── mcp/          registry.go — hardcoded 5 MCP server definitions
│   └── web/          server.go, handlers.go, sse.go, handlers_test.go
├── pipelines/        *.yaml — runtime pipeline store (not embedded)
└── run_history.json  — runtime ROI log (created on first run)
```

## Key design rules

- **No `go:embed` inside `internal/`** — all embed declarations live in `assets/assets.go`. Sub-packages accept `fs.FS` in constructors.
- **One Qwen process per step** — `qwen.Client.Run` spawns and waits; no shared subprocess.
- **SSE not WebSocket** — all streaming uses Server-Sent Events via `notify.Hub`.
- **`ApprovalHub` is shared** — both approval gates and repair proposals use the same `chan bool` mechanism keyed by `runID:stepID` (repair uses `runID:stepID:repair`).
- **`StartAsync` vs `Execute`** — handlers call `StartAsync(runID, p, params)` to get an ID before the run starts; `Execute` is for synchronous use (tests).

## How to add a new skill

1. Create `assets/skills/<your-skill-name>.md` with an `# Skill: <name>` header, a `## Parameters` section, and `## Instructions` steps.
2. The skill is automatically available in the creator sidebar and via `/api/skills`.
3. Reference it in a pipeline step: `type: skill`, `skill: <your-skill-name>`.

## How to add a new MCP server

1. Add a new `Server` entry in `internal/mcp/registry.go` inside `NewRegistry()`.
2. Add at least one `Tool` entry with `Name` and `Description`.
3. The server is automatically available in `/api/mcps` and the creator sidebar.
4. Reference it in a pipeline step: `type: mcp`, `server: <name>`, `tool: <tool-name>`.

## How to add a new template

1. Create `assets/templates/<name>.yaml` with a valid pipeline YAML (must have `name`, `steps`, `description`, `manual_minutes`).
2. The template appears automatically in the `/templates` gallery and via `/api/templates`.
3. Run `go build ./...` to verify the embed compiles cleanly.

## Runner event lifecycle

```
running → narration* → success
running → narration* → failed
         └─ repair_suggested → (approved) → running → success
                              → (rejected) → failed → postmortem
running → awaiting_approval → (approved) → next step
                             → (rejected/timeout) → failed
```

All events are `notify.StepEvent` structs serialized as JSON in the SSE stream.

## SberChat integration (stub)

`internal/notify/sberchat.go` has a `TODO(team)` comment. To implement:
- Fill in `BaseURL`, `ChatID`, `Token` in the struct.
- Implement `Notify` as a POST to `BaseURL/messages` with `{"chat_id": ChatID, "text": formatEvent(event)}` and `Authorization: Bearer Token` header.
- Wire it into `MultiNotifier` in `cmd/wzr/main.go`.

## Commit message rules

- One short sentence only, no body, no trailers.
- No `Co-Authored-By` or AI attribution.
- Mark plan checkboxes `[x]` in `docs/plans/20260711-wzr-mvp.md` and include the plan file in the same commit.
