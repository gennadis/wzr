# WZR — WZR's Zen Runtime

Pipeline orchestrator for Sber internal infrastructure. Single Go binary — backend, web UI, embedded skills and templates.

## What it does

- **Pipeline creator** — build pipelines step-by-step with AI chat or describe your goal and generate a full YAML
- **Execution dashboard** — run pipelines, watch live narration as Qwen works, approve gates, handle failures
- **Self-healing** — when a step fails, AI diagnoses the error, proposes a fix, you approve, pipeline continues
- **Templates gallery** — 4 pre-built templates: release manager, PR review, incident response, onboarding
- **ROI counter** — tracks time saved across all runs

## Build

```sh
go build -o wzr ./cmd/wzr
```

Requires Go 1.22+. No other dependencies needed at build time.

## Run

```sh
./wzr
```

Open [http://localhost:8080](http://localhost:8080).

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8080` | HTTP listen port |
| `--qwen` | `qwen` | Path to Qwen Code CLI binary |
| `--pipelines` | `./pipelines` | Directory for pipeline YAML files |
| `--history` | `./run_history.json` | Path to run history JSON file |
| `--dry-run <name>` | — | Parse named pipeline, print it, exit |

### Dry-run example

```sh
./wzr --dry-run release-manager --pipelines ./pipelines
# Pipeline: release-manager (v1.0)
# Steps: 6
#   [skill] Check Jira tickets readiness (check-readiness)
#   ...
```

## Execution engine

WZR ships two `CLIExecutor` presets. Use `--qwen` to point at the binary if it is not on `$PATH`.

| Preset | Binary | Model |
|--------|--------|-------|
| `NewQwen()` (default) | `qwen` | `deepseek-v4-flash` |
| `NewGigacode()` | `gigacode` | `vllm/Qwen3-Coder-Next-262k` |

Each pipeline step runs in agentic mode:

```sh
qwen -p "<step prompt>" --output-format text --approval-mode auto-edit --allowed-tools run_shell_command --model deepseek-v4-flash
```

Creator and post-run chat use text-only mode (no tools):

```sh
qwen -p "<prompt>" --output-format text --allowed-tools ""
```

## Pipeline YAML format

```yaml
name: my-pipeline
version: "1.0"
description: What this pipeline does
manual_minutes: 60   # drives ROI counter
params:
  project: ""
  version: ""
steps:
  - id: check-readiness
    name: Check Jira tickets
    type: skill                    # skill | mcp | approval
    skill: check-release-readiness
    params:
      project: "{{ .project }}"
      version: "{{ .version }}"

  - id: trigger-build
    name: Trigger Jenkins build
    type: mcp
    server: jenkins
    tool: trigger_build
    params:
      job: "release/{{ .project }}"

  - id: confirm-deploy
    name: Confirm deployment
    type: approval
    timeout_minutes: 30
```

## Creator walkthrough

1. Go to **Creator** → **Describe & Generate** tab
2. Type: *"Check Jira readiness, verify PRs, trigger Jenkins build, get human approval, update Confluence"*
3. Click **Generate YAML** — Qwen builds the pipeline YAML
4. Enter a pipeline name and click **Save Pipeline**
5. Go to **Dashboard** — your pipeline appears there, ready to run

## Dashboard walkthrough

1. Open **Dashboard** — see all your pipelines
2. Click **Run** on a pipeline → fill in params → **Confirm**
3. Watch the **Live Narration** sidebar as Qwen executes each step
4. On **Approval** steps: click **Approve** or **Reject**
5. On **failure**: amber repair card appears with AI diagnosis → **Apply & Retry** or **Give Up**
6. After run: ask questions in **Post-run Chat**

## MCP servers

WZR ships with 5 pre-configured MCP servers:

| Server | Tools |
|--------|-------|
| `bitbucket` | `list_pull_requests`, `get_pr`, `merge_pr` |
| `jira` | `search_issues`, `transition_issue`, `get_issue` |
| `jenkins` | `trigger_build`, `get_build_status`, `list_jobs` |
| `confluence` | `get_page`, `create_page`, `update_page` |
| `postgres` | `query`, `list_tables` |

## Embedded skills

| Skill | Purpose |
|-------|---------|
| `check-release-readiness` | Checks Jira tickets are in "Done"/"Ready" before releasing |
| `update-release-notes` | Updates Confluence release notes page from Jira issues |

## Notifications

WZR fires all run events through a `Notifier`. Currently only `SSENotifier` is wired in `main.go`:

- **SSENotifier** — streams all events to the dashboard in real time via Server-Sent Events; includes a replay buffer so late subscribers receive events emitted before they connected
- **MultiNotifier** — fan-out wrapper available in `internal/notify/multi.go`; wire it in `main.go` when adding additional notifiers
- **SberChatNotifier** — stub; `TODO(team)` in `internal/notify/sberchat.go` to implement the REST call

## Development

```sh
go mod tidy && go mod verify
go build ./...
go test ./...
go vet ./...
golangci-lint run ./...
```

All six must pass before committing.
