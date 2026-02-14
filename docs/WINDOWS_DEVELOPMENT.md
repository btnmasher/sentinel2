# Windows Development

Native Windows development is supported for the main backend/frontend workflow without requiring WSL, Cygwin, or Git Bash.

## Native Windows Setup

Install:
1. Go 1.25+
2. Task 3.x
3. Bun 1.3+

Recommended:
- Keep the repo on a local filesystem path (for example `C:\Code\sentinel2`).
- Use Windows Terminal.

## Core Workflow (Native Windows)

```powershell
task setup
task dev
```

`task dev` uses the cross-platform `taskutil` supervisor and provides:
- frontend/backend split-pane TUI
- restart/rebuild controls
- migration shortcut
- log session files

### Dev supervisor keybinds
- `q` / `Ctrl+C`: quit
- `tab`, `1`, `2`: pane focus
- `r`: restart focused process
- `f` / `b`: restart frontend/backend
- `F`: rebuild frontend and restart frontend
- `R`: rebuild backend and restart backend
- `m`: run migrate then restart backend

## Useful Commands

- `task build`
- `task build:migrate`
- `task dev:migrate`
- `task dev:logs`
- `task dev:logs:clean`

## Unix-Only Tasks

These tasks are intentionally Unix-only and guarded in Taskfile:
- `task dev:logs:view`
- `task dev:logs:view:json`
- `task completion:install`

## Optional Tools

- `golangci-lint`:
  - `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`

## Notes

- Docker-based workflows are still available on Windows via Docker Desktop.
- Release asset generation remains CI-focused.
