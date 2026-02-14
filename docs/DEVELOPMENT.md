# Development

This document describes local development, Docker workflows, CI flows, and the current root Task command set.

## Dependencies

Requirements:
- Go **1.25+**
- Bun **1.3+**
- Task **3.x**

Install these once on a new machine:
1. Go  
   Download: [go.dev/dl](https://go.dev/dl/)
2. Task
```bash
go install github.com/go-task/task/v3/cmd/task@latest
```
3. Bun
```bash
curl -fsSL https://bun.com/install | bash
```

## Windows-Native Quickstart

For native Windows development (no WSL/Cygwin/Git Bash required):

```powershell
task setup
task dev
```

See `docs/WINDOWS_DEVELOPMENT.md` for the full Windows notes.

## Local Dev (Host)

### `task setup`
- bootstrap/preflight command
- checks tool versions
- installs frontend deps
- downloads backend Go modules

### `task dev`
- builds backend without embedded frontend (`task build:backend:skip-embed`)
- runs `taskutil` supervisor with:
  - frontend and backend side-by-side TUI panes
  - keybinds for restart/rebuild/migrate actions
  - session log files (`vite.log`, `backend.log`, `backend.jsonl`)

Keybinds:
- `q` or `Ctrl+C`: quit
- `tab`, `1`, `2`: pane focus
- `r`: restart focused process
- `f` / `b`: restart frontend/backend
- `F`: rebuild frontend and restart frontend
- `R`: rebuild backend and restart backend
- `m`: run migrate then restart backend

### Common local entrypoints
- `task build` -> frontend deps + frontend build + backend build
- `task build:migrate` -> build everything + run migrations
- `task build:migrate:skip-embed` -> build backend only + run migrations
- `task lint` -> backend + frontend lint

## Local Dev with Docker

### `task docker:dev:up`
- runs `docker compose -f docker-compose.dev.yml up --build`
- starts two services:
  - `frontend-dev` (`bun run dev`)
  - `backend-dev` (backend serve with `DEV_PROXY=frontend-dev:5173`)

### `task docker:dev:up:migrate`
- runs migrations in `backend-dev` first
- then starts dev compose

### Other Docker dev tasks
- `task docker:dev:down`
- `task docker:dev:logs`
- `task docker:dev:status`

## Docker Production Build

### `task docker:build`
- `docker build --build-arg BUILD_VERSION=... -t sentinel2 .`

### Dockerfile stages
- `toolchain`: installs Go/Bun/Task and base packages
- `deps`: caches backend/frontend dependencies
- `build`: runs `task build`
- runtime: ships `bin/sentinel2-server` in slim image

## CI

### Lint workflow (`.github/workflows/lint.yml`)
- backend lint job
- frontend lint job
- uploader lint job (runs from `uploader/` taskfile)

### Release workflow (`.github/workflows/release.yml`)
- trigger: tag push `v*`
- setup Go + Bun
- download backend modules + install frontend deps
- build frontend and stage embed assets
- cross-compile backend binaries directly with `go build`
- publish `dist/*.zip`
- build and push GHCR image tags (`<tag>`, `latest`)

## Migrations

Database migrations live in `backend/pb_migrations` and are compiled into the backend binary.

- migration sources: `backend/pb_migrations`
- Rebuild the server binary after changing migration files.
- Create a migration with:
```bash
task migrate:create NAME=your_migration_name
```
- Run migrations locally with:
```bash
task build:migrate
```
PocketBase migration authoring reference:  
https://pocketbase.io/docs/go-migrations/

## Versioning

Build versions are derived from Git tags, branch, and working tree state.

Base tag selection:

| Condition | Base version |
| --- | --- |
| Exact tag on HEAD | `vX.Y.Z` |
| No exact tag, but tags exist | `vX.Y.Z` |
| No tags at all | `v0.0.0` |

Main branch (`main`):

| Condition | Version |
| --- | --- |
| Exact tag, clean | `vX.Y.Z` |
| Exact tag, dirty | `vX.Y.Z-dev` |
| Not on tag, tags exist, clean | `vX.Y.Z-<shortsha>` |
| Not on tag, tags exist, dirty | `vX.Y.Z-<shortsha>-dev` |
| No tags, clean | `v0.0.0-<shortsha>` |
| No tags, dirty | `v0.0.0-<shortsha>-dev` |

Other branches (`feature/foo`):

| Condition | Version |
| --- | --- |
| Exact tag, clean | `vX.Y.Z-branch-feature-foo` |
| Exact tag, dirty | `vX.Y.Z-dev-branch-feature-foo` |
| Not on tag, tags exist, clean | `vX.Y.Z-branch-feature-foo` |
| Not on tag, tags exist, dirty | `vX.Y.Z-dev-branch-feature-foo` |
| No tags, clean | `v0.0.0-branch-feature-foo` |
| No tags, dirty | `v0.0.0-dev-branch-feature-foo` |

Detached HEAD:

| Condition | Version |
| --- | --- |
| Exact tag, clean | `vX.Y.Z` |
| Exact tag, dirty | `vX.Y.Z-dev` |
| Not on tag, tags exist, clean | `vX.Y.Z-<shortsha>` |
| Not on tag, tags exist, dirty | `vX.Y.Z-<shortsha>-dev` |
| No tags, clean | `v0.0.0-<shortsha>` |
| No tags, dirty | `v0.0.0-<shortsha>-dev` |

## Notes

- `task setup` checks and downloads frontend/backend dependencies.
- Typical day-to-day dev loop: `task dev`, and run `task setup` when dependencies/toolchain state changes.
- `task setup:taskutil` builds the helper binary used by cross-platform dev helper tasks.
- Unix-only tasks are guarded with `platforms` in Taskfile:
  - `task dev:logs:view`
  - `task dev:logs:view:json`
  - `task completion:install`

## Task Command Index

| Command | What it does |
| --- | --- |
| `task build` | Build frontend and backend. |
| `task clean` | Remove local build artifacts and tool caches (preserves bin/.env and bin/pb_data). |
| `task default` | Build frontend and backend. |
| `task dev` | Run Vite dev server + backend. |
| `task ensure-deps` | Ensure Bun build dependency is available. |
| `task lint` | Run backend and frontend lint. |
| `task run` | Run the backend server (expects bin/sentinel2-server). |
| `task setup` | Verify Go/Bun and install frontend/backend dependencies. |
| `task build:backend` | Build backend binary with embedded frontend. |
| `task build:backend:darwin` | Build backend binary for macOS (arm64). |
| `task build:backend:linux` | Build backend binary for Linux (amd64). |
| `task build:backend:skip-embed` | Build backend binary without embed tag. |
| `task build:backend:windows` | Build backend binary for Windows (amd64). |
| `task build:frontend` | Build frontend bundle. |
| `task build:frontend:dev` | Build frontend bundle without minification. |
| `task build:migrate` | Build everything and run migrations. |
| `task build:migrate-run` | Build, run migrations, and start the server. |
| `task build:migrate:skip-embed` | Build backend without embed assets and run migrations. |
| `task build:run` | Build everything and start the server. |
| `task completion:install` | Unix-only: install task shell completion for the current shell. |
| `task dev:backend` | Run backend with DEV_PROXY set. |
| `task dev:frontend` | Run Vite dev server. |
| `task dev:logs` | Tail Vite and backend dev logs. |
| `task dev:logs:clean` | Delete dev log folders older than KEEP_DAYS (default 7). |
| `task dev:logs:view` | Unix-only: open dev logs in lnav (use task dev:logs on Windows). |
| `task dev:logs:view:json` | Unix-only: open JSON log file in lnav (use task dev:logs on Windows). |
| `task dev:migrate` | Run dev workflow with migrations first. |
| `task dev:stop` | Show how to stop dev. |
| `task docker:build` | Build Docker image with version tags. |
| `task docker:dev:down` | Stop dev compose. |
| `task docker:dev:logs` | Tail dev compose logs. |
| `task docker:dev:status` | Show dev compose status. |
| `task docker:dev:up` | Start dev compose (Vite + backend). |
| `task docker:dev:up:migrate` | Run migrations, then start dev compose. |
| `task docker:down` | Stop production compose. |
| `task docker:logs` | Tail production compose logs. |
| `task docker:restart` | Alias for docker:up:detach. |
| `task docker:restart:migrate` | Alias for docker:up:migrate:detach. |
| `task docker:status` | Show production compose status. |
| `task docker:up` | Start production compose. |
| `task docker:up:detach` | Start production compose in detached mode. |
| `task docker:up:migrate` | Run migrations, then start production compose. |
| `task docker:up:migrate:detach` | Run migrations, then start production compose in detached mode. |
| `task ensure-deps:bun` | Ensure Bun temp/cache directories exist. |
| `task go:toolchain:update` | Update module toolchain to latest Go (requires Go 1.21+). |
| `task lint:backend` | Run backend lint. |
| `task lint:backend:fix` | Run backend lint with auto-fix (local only). |
| `task lint:frontend` | Run frontend lint. |
| `task lint:frontend:fix` | Run frontend lint with auto-fix (local only). |
| `task migrate:run` | Run database migrations (expects bin/sentinel2-server). |
| `task migrate:create` | Create a new PocketBase migration (NAME required). |
| `task prepare:embed` | Copy frontend dist to embed location. |
| `task release:assets` | Build release assets locally (backend binaries); CI release workflow builds directly without Task. |
| `task setup:deps:backend` | Download backend Go module dependencies. |
| `task setup:deps:frontend` | Install frontend dependencies. |
| `task setup:taskutil` | Build taskutil helper binary. |
| `task setup:version:bun` | Print Bun version using project-local Bun cache dirs. |

## Utilities

Task completion:
```bash
task completion:install
```

Optional tooling:
- `lnav` for `task dev:logs:view` and `task dev:logs:view:json`
- `golangci-lint` v2 for backend lint
