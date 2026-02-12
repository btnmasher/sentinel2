# Development

This document describes the current command/task execution trees for local development, Docker development, Docker production builds, and CI.

## Dependencies

Requirements:
- Go **1.25+**
- Bun **1.3+**
- Task **3.x**
- Zig (required for uploader app cross-compilation tasks)
- `zip` CLI (Info-ZIP) for uploader archive tasks

Install these once on a new machine:
1. Go (required for backend/uploader builds and Taskfile install).
   Download: [go.dev/dl](https://go.dev/dl/)
Linux (tarball install example):
```bash
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf goX.Y.Z.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```
Toolchain management (required for `go get go@latest`) needs Go 1.21+.
2. Task:
```bash
go install github.com/go-task/task/v3/cmd/task@latest
```
3. Bun (frontend runtime + tooling).
macOS/Linux:
```bash
curl -fsSL https://bun.com/install | bash
```
4. `zip` CLI (required for `task build:uploader` and `task release:assets`).
- Debian/Ubuntu: `sudo apt-get install zip`
- Fedora/RHEL: `sudo dnf install zip`
- macOS: preinstalled on most systems (or `brew install zip`)
- Windows users: see [WINDOWS_DEVELOPMENT.md](WINDOWS_DEVELOPMENT.md)
5. Zig (required for `task build:uploader:windows`; used by CI/Docker cross-compilation and optional local Linux Zig builds via `task build:uploader:linux:zig`).
   Download: [ziglang.org/download](https://ziglang.org/download/)

## Local Dev (Host)

### `task setup`
- Use this as a bootstrap/preflight command (new machine, fresh clone, CI prep).
- It is not required before every `task dev` run.
- `task ensure-deps:bun`
- `go version` / `bun --version`
- Frontend dependencies:
  - `cd frontend && bun install --frozen-lockfile`
- Backend dependencies:
  - `cd backend && go mod download`
- Uploader dependencies:
  - `cd uploader && go mod download`

### `task dev` (tmux workflow)
- `task ensure-deps:bun`
- `task build:backend:skip-embed`
- Starts tmux session (`sentinel2-dev`) with two panes:
  - Frontend pane:
    - frontend `bun install`
    - `bun run dev`
  - Backend pane:
    - optional migrate when `DEV_MIGRATIONS=1`
    - `./bin/sentinel2-server serve --dev`

### `task dev:plain`
- `task ensure-deps:bun`
- `task build:backend:skip-embed`
- Starts frontend dev server in background
- Runs backend serve in foreground

### Note
- `task dev` and `task dev:plain` both start backend with `DEV_PROXY` (default `127.0.0.1:5173`) so frontend hot-reload remains active through the backend-origin development flow.

### `task dev:with-uploader`
- `task setup`
- `task build:uploader`
  - validates `zip` availability
  - Linux uploader build:
    - default: `task build:uploader:linux`
    - optional: `UPLOADER_LINUX_TOOLCHAIN=zig task build:uploader` -> `task build:uploader:linux:zig`
  - Windows uploader build:
    - `task build:uploader:windows` (depends on `task ensure-deps:zig`)
  - macOS uploader build:
    - `task build:uploader:darwin` -> `task build:uploader:darwin:headless`
  - Creates stable zip artifacts in `frontend/public/downloads/`
- `task dev`

### Related local entrypoints
- `task dev:migrate` -> `DEV_MIGRATIONS=1 task dev`
- `task build`:
  - `task setup`
  - `task build:uploader`
  - `task build:frontend`
  - `task build:backend`

## Local Dev with Docker

This mode uses a bind mount of your local repo into the container (`./:/app`), so source changes on host are visible immediately in the container process for Vite/backend development.

### `task docker:dev:up`
- `docker compose -f docker-compose.dev.yml up --build`
- Service command: `task dev:container`
  - `task ensure-deps:bun`
  - frontend `bun install` + `bun run dev`
  - `task build:backend:skip-embed`
  - backend `serve --dev`

### `task docker:dev:up:migrate`
- `docker compose -f docker-compose.dev.yml run --rm sentinel2-dev task build:migrate:skip-embed`
- `docker compose -f docker-compose.dev.yml up --build`

### Other Docker dev tasks
- `task docker:dev:down`
- `task docker:dev:logs`
- `task docker:dev:status`

## Docker Production/Release Build (Local)

### `task docker:build`
- `docker build --build-arg BUILD_VERSION=... -t sentinel2 .`

### Dockerfile stage flow
- `toolchain` stage:
  - installs build toolchain: Zig, Bun, Task, base packages
- `deps` stage:
  - caches Go module downloads (backend/uploader)
  - caches frontend dependency install
- `build` stage:
  - copies source
  - runs `task build`
- runtime stage:
  - copies `bin/sentinel2-server` into slim image

### Runtime compose tasks
- `task docker:up`
- `task docker:up:detach`
- `task docker:up:migrate`
- `task docker:up:migrate:detach`
- `task docker:restart`
- `task docker:restart:migrate`

## CI Build/Lint/Release

### Lint Workflow (`.github/workflows/lint.yml`)

#### Backend lint job
- checkout
- setup Task + Go
- restore Go and lint caches
- install `golangci-lint`
- run `task lint:backend`

#### Frontend lint job
- checkout
- setup Task + Bun
- restore Bun caches
- run `task lint:frontend`

### Release Workflow (`.github/workflows/release.yml`)

- trigger: push tag `v*`
- setup Task, Go, Bun, Zig
- restore Go/Bun/Zig caches
- run `task setup`
- run `task release:assets`
  - `task build:uploader`
  - `task build:frontend`
  - cross-build backend artifacts (linux/darwin/windows) via `task build:backend`
  - collect outputs into `dist/`
- publish `dist/*` as GitHub release artifacts

## Migrations

Database migrations live in `backend/pb_migrations` and are compiled into the backend binary.

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
- Use `task ensure-deps` to quickly verify Bun + Zig availability for build workflows that need both.
- `task setup` checks and downloads frontend/backend/uploader dependencies (except zig).
- Zig is required by Zig-dependent uploader builds (`task build:uploader:windows`, optional `task build:uploader:linux:zig`).
- In practice, Zig is primarily needed for CI/Docker release and cross-compilation flows; local Linux development can use the default native uploader build path.
- Typical day-to-day dev loop: `task dev` (or `task dev:plain`), and run `task setup` only when dependencies/toolchain state changes.

## Task Command Index

| Command | What it does |
| --- | --- |
| `task build` | Build uploader, frontend, and backend. |
| `task clean` | Remove local build artifacts and tool caches (preserves bin/.env and bin/pb_data). |
| `task default` | Build uploader, frontend, and backend. |
| `task dev` | Run Vite dev server + backend (no uploader build). |
| `task ensure-deps` | Ensure Bun and Zig build dependencies are available. |
| `task lint` | Run backend and frontend lint. |
| `task run` | Run the backend server (expects bin/sentinel2-server). |
| `task setup` | Verify Go/Bun and install frontend/backend/uploader dependencies. |
| `task build:backend` | Build backend binary with embedded frontend. |
| `task build:backend:darwin` | Build backend binary for macOS (amd64). |
| `task build:backend:linux` | Build backend binary for Linux (amd64). |
| `task build:backend:skip-embed` | Build backend binary without embed tag. |
| `task build:backend:windows` | Build backend binary for Windows (amd64). |
| `task build:frontend` | Build frontend bundle. |
| `task build:frontend:dev` | Build frontend bundle without minification. |
| `task build:migrate` | Build everything and run migrations. |
| `task build:migrate-run` | Build, run migrations, and start the server. |
| `task build:migrate:skip-embed` | Build backend without embed assets and run migrations. |
| `task build:run` | Build everything and start the server. |
| `task build:uploader` | Build uploader bundles (set UPLOADER_LINUX_TOOLCHAIN=zig for Zig Linux build). |
| `task build:uploader:darwin` | Build uploader for macOS (amd64, headless tag). |
| `task build:uploader:darwin:headless` | Build headless uploader for macOS (amd64). |
| `task build:uploader:headless` | Build headless uploader binaries for Linux, Windows, and macOS (amd64). |
| `task build:uploader:linux` | Build uploader for Linux (amd64). |
| `task build:uploader:linux:headless` | Build headless uploader for Linux (amd64). |
| `task build:uploader:linux:zig` | Build uploader for Linux (amd64) using Zig as CC/CXX. |
| `task build:uploader:windows` | Build uploader for Windows (amd64). |
| `task build:uploader:windows:headless` | Build headless uploader for Windows (amd64). |
| `task completion:install` | Install task shell completion for the current shell. |
| `task dev:attach` | Attach to the dev tmux session. |
| `task dev:backend` | Run backend with DEV_PROXY set. |
| `task dev:container` | Run Vite + backend in a dev container. |
| `task dev:frontend` | Run Vite dev server. |
| `task dev:logs` | Tail Vite and backend dev logs. |
| `task dev:logs:clean` | Delete dev log folders older than KEEP_DAYS (default 7). |
| `task dev:logs:view` | Open dev logs in lnav (requires lnav). |
| `task dev:logs:view:json` | Open JSON log file in lnav (requires LOG_JSON or LOG_JSON_PATH). |
| `task dev:migrate` | Run dev workflow with migrations first. |
| `task dev:plain` | Run dev workflow without tmux. |
| `task dev:stop` | Stop the dev tmux session. |
| `task dev:tmux` | Run Vite + backend in a tmux session. |
| `task dev:with-uploader` | Build uploader bundles, then run Vite + backend. |
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
| `task ensure-deps:zig` | Ensure zig is installed. |
| `task go:toolchain:update` | Update module toolchain to latest Go (requires Go 1.21+). |
| `task lint:backend` | Run backend lint. |
| `task lint:backend:fix` | Run backend lint with auto-fix (local only). |
| `task lint:frontend` | Run frontend lint. |
| `task lint:frontend:fix` | Run frontend lint with auto-fix (local only). |
| `task migrate:create` | Create a new PocketBase migration (NAME required). |
| `task prepare:embed` | Copy frontend dist to embed location. |
| `task release:assets` | Build release assets (backend binaries + uploader zips). |

## Developer Utilities

Task completion:
```bash
task completion:install
```

Manual completion fallback:
```bash
task --completion bash
task --completion zsh
task --completion fish
task --completion powershell
```

Bash install example:
```bash
task --completion bash > ~/.local/share/bash-completion/completions/task
```

Optional tooling:
- `lnav` for `task dev:logs:view` and `task dev:logs:view:json`
- `golangci-lint` v2 for backend lint
