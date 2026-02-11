# Sentinel 2

Sentinel 2 is an EVE intel and navigation suite that pairs a realtime intel feed with interactive region maps, route planning, and a companion uploader for EVE chat logs. It supports both standalone EVE OAuth and TEST Auth (OIDC) deployments, with admin/staff tooling for moderation and account management.

## Architecture
- Single-binary backend with embedded frontend assets for simple deployment.
- PocketBase powers data storage, auth sessions, realtime events, and admin tooling.
- Frontend is served by the backend in production; Vite dev server can be proxied.
- Uploader is a separate Go binary that posts intel logs to the server.

## Tech Stack
- **Backend:** Go, PocketBase, SQLite (via PocketBase), EVE ESI integration.
- **Frontend:** React, Vite, Tailwind CSS, daisyUI, Zustand, React Router.

### Docs
These docs capture the structure, conventions, and development practices for each side of the codebase.

- [FRONTEND.md](FRONTEND.md)
- [BACKEND.md](BACKEND.md)
- [WINDOWS_DEVELOPMENT.md](WINDOWS_DEVELOPMENT.md)

## Key Capabilities
- Realtime intel feed with filters and alarms.
- Region map with routes, waypoints, jump bridge links, and multiple layouts.
- Dedicated navigation page for quick route planning.
- Uploader companion tool with per-account tokens and cross-platform downloadable bundles.
- Admin/staff views for moderation, account management, and audits.

## Auth Modes
- **EVE OAuth (standalone):** Character linking, profile management, admin tools.
- **TEST Auth (OIDC):** Uses OIDC claims and roles; account/character management is handled externally (no in-app character linking).

## Requirements
- Go **1.25+**
- Bun **1.3+**
- Task **3.x**
- `zip` CLI (Info-ZIP) for uploader archive tasks

## Dependencies
Install these once on a new machine:
1. Go (required for backend/uploader builds and Taskfile install).
   Download: [go.dev/dl](https://go.dev/dl/)
Linux (tarball install example):
```
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf goX.Y.Z.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```
Toolchain management (required for `go get go@latest`) needs Go 1.21+.
2. Task:
```
go install github.com/go-task/task/v3/cmd/task@latest
```
3. Bun (frontend runtime + tooling).
macOS/Linux:
```
curl -fsSL https://bun.com/install | bash
```
4. `zip` CLI (required for `task build:uploader` and `task release:assets`).
- Debian/Ubuntu: `sudo apt-get install zip`
- Fedora/RHEL: `sudo dnf install zip`
- macOS: preinstalled on most systems (or `brew install zip`)
- Windows users: see [WINDOWS_DEVELOPMENT.md](WINDOWS_DEVELOPMENT.md).

## Environment (Backend)
Minimum set (extend as needed):
```
AUTH_BACKEND=eve|testauth
OIDC_CLIENT_ID=
OIDC_CLIENT_SECRET=
OIDC_SCOPES=openid
OIDC_REQUIRED_ROLES=urn:sso:alliance:test-alliance,urn:sso:allies
OIDC_STAFF_ROLES=urn:sso:staff_user
OIDC_USERINFO_URL=https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/userinfo
ESI_DIRECT_BASE_URL=https://esi.evetech.net/latest/
ESI_PROXY_BASE_URL=https://auth.pleaseignore.com/esi/
```

## Local Development (Manual)
The Taskfile is the preferred workflow, but you can run things manually if needed.

Backend:
```
cd backend
go run . migrate up
go run . --dev
```

Frontend:
```
cd frontend
bun install
bun run dev
```

The backend serves on `:8090` by default.

To use the backend as a reverse proxy for the Vite dev server (so auth cookies stay on one origin):
```
./bin/sentinel2-server serve --dev-proxy 127.0.0.1:5173
```

## Build (local)
This builds the uploader zips, the frontend, and the backend binary.
```
task build
```

Artifacts:
- `bin/sentinel2-server` (server binary)
- `frontend/public/downloads/*.zip` (uploader bundles)

Local dev:
- `task dev` runs Vite + backend in tmux (tmux required).
- `task dev:plain` runs Vite + backend without tmux.
- `task dev:with-uploader` rebuilds uploader bundles first, then runs dev.
- `task dev:migrate` runs migrations before starting dev servers.
- `task dev:tmux` aliases `task dev`.
- `task dev:migrate:tmux` runs migrations and uses tmux.
- `task dev:attach` attaches to the existing dev tmux session.
- `task dev:stop` stops the dev tmux session.
- `task dev:logs` tails dev log files (Vite + backend) from the latest session folder in `/tmp/sentinel2-dev`.
- `task dev:logs:view` opens the latest session logs in `lnav` (lnav required).
- `task dev:logs:view:json` opens JSONL logs in `lnav` (requires `LOG_JSON=true`).
- `task dev:logs:clean` deletes dev log folders older than `KEEP_DAYS` (default 7).
Override defaults with env vars:
```bash
VITE_HOST=0.0.0.0 VITE_PORT=5174 task dev
DEV_PROXY=127.0.0.1:5174 task dev
```
Note: dev mode runs the backend binary from `bin/`, so `.env` and `pb_data` can live there.

Typical workflow:
1. Install dependencies (Go, Task, Bun).
2. Run `task setup` to install frontend deps and Go modules.
3. Run `task dev` for local Vite + backend.
4. If needed, run `task build:migrate` (or `task build:migrate-run`) to apply migrations.
5. Use `task build` when you need embedded frontend + uploader bundles.
6. Run `task lint` before opening a PR.
7. Use `task migrate:create NAME=your_migration` when adding new migrations. The name is used in the migration filename; use `snake_case` (e.g., `add_intel_index`).
8. Run `task completion:install` to install Task autocompletion for your current shell.

Task autocompletion:
```bash
task completion:install
```

Manual fallback:
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

Optional dev tools:
- `lnav` (optional) for `task dev:logs:view` and `task dev:logs:view:json`.
- `golangci-lint` (required for `task lint:backend`).

Install lnav:
- macOS: `brew install lnav`
- Fedora/RHEL: `sudo dnf install lnav`
- Debian/Ubuntu: `sudo apt-get install lnav`
- Windows: see [WINDOWS_DEVELOPMENT.md](WINDOWS_DEVELOPMENT.md).

Install golangci-lint:
- Recommended (binary install): `curl -sSfL https://golangci-lint.run/install.sh | sh -s v2.9.0`
- macOS (brew): `brew install golangci-lint`
- Windows: see [WINDOWS_DEVELOPMENT.md](WINDOWS_DEVELOPMENT.md).
- Go install (not recommended by upstream): `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
Note: upstream recommends binary installs and discourages `go install` for golangci-lint.

## Logging
The backend uses PocketBase's built-in `slog` logger, so logs show up in the PocketBase Logs UI. Request logs include a `request_id` plus `type`, `status`, and `duration_ms` fields. Domain logs (auth, intel, map, SDE, cleanup) use structured fields like `type` to make filtering easier.
View logs in the PocketBase Admin UI under **Logs**.
Optional verbose debugging (prints SQL + debug logs):
- `DEBUG_ENABLED=true` (or pass `--dev` to the server)
Optional log settings:
- `LOG_LEVEL=debug|info|warn|error`
- `LOG_PRETTY=true` (prints a styled console view for app logs)
- `LOG_PRETTY_PB=true` (experimental: replaces PocketBase dev console printer using go:linkname)
- `LOG_JSON=true` (write JSONL logs for app output)
- `LOG_JSON_PATH=/path/to/backend.jsonl`
- `LOG_JSON_PB=true` (experimental: also write PocketBase dev console logs to JSONL)

## Docker
The Docker build cross‑compiles uploader binaries and embeds them into the frontend build.
```
docker build -t sentinel2 .
docker run -p 8090:8090 -v pb_data:/app/pb_data sentinel2
```

Or with compose:
```
docker compose up --build
```

Useful commands:
- `task docker:build` build the image locally with version tagging.
- `task docker:up` start production compose.
- `task docker:up:migrate` run migrations, then start production compose.
- `docker run -p 8090:8090 -v pb_data:/app/pb_data sentinel2` run with a named volume.
- `docker compose up --build` build and run via compose.
- `docker compose down` stop services (data remains in the volume).
Set `DOCKER_IMAGE=sentinel2:dev` to control the tag used by Taskfile.
Compose reads `.env` (via `env_file`) to populate container environment variables.

Dev compose (Vite + backend in one container):
```
task docker:dev:up
```
This runs Vite on `:5173` and the backend on `:8090` with `DEV_PROXY` set.
Use `task docker:dev:up:migrate` to run migrations on startup.

## Data Storage
PocketBase data is stored in `pb_data` (mounted via Docker volume in production).

## Migrations
Database migrations live in `backend/pb_migrations` and are compiled into the backend binary. Rebuild the server binary after changing any migration files. For migration authoring details, see the PocketBase Go migrations docs:
```
https://pocketbase.io/docs/go-migrations/
```

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




## Build + Lint Command Table
| Command | What it does |
| --- | --- |
| `task build` | Build uploader + frontend + backend. |
| `task clean` | Remove local build artifacts and tool caches. |
| `task build:frontend` | Build frontend bundle. |
| `task build:frontend:dev` | Build frontend without minification. |
| `task build:backend` | Build backend with embedded frontend. |
| `task build:backend:linux` | Build backend for Linux (amd64). |
| `task build:backend:darwin` | Build backend for macOS (amd64). |
| `task build:backend:windows` | Build backend for Windows (amd64). |
| `task build:backend:skip-embed` | Build backend without embed tag. |
| `task build:uploader` | Build uploader bundles. |
| `task build:uploader:linux` | Build uploader for Linux (amd64). |
| `task build:uploader:darwin` | Build uploader for macOS (amd64). |
| `task build:uploader:windows` | Build uploader for Windows (amd64). |
| `task build:run` | Build everything and start server. |
| `task run` | Run the backend server (expects bin/sentinel2-server). |
| `task build:migrate` | Build everything and run migrations. |
| `task build:migrate-run` | Build, run migrations, start server. |
| `task dev` | Run Vite dev server + backend (no uploader build). |
| `task dev:plain` | Run dev workflow without tmux. |
| `task dev:with-uploader` | Build uploader bundles, then run Vite + backend. |
| `task dev:migrate` | Run dev workflow with migrations first. |
| `task dev:tmux` | Alias for `task dev`. |
| `task dev:migrate:tmux` | Run dev workflow with migrations in tmux. |
| `task dev:attach` | Attach to the dev tmux session. |
| `task dev:stop` | Stop the dev tmux session. |
| `task dev:logs` | Tail dev log files. |
| `task dev:logs:view` | Open dev logs in lnav. |
| `task dev:logs:view:json` | Open JSON log file in lnav. |
| `task dev:logs:clean` | Delete dev log folders older than KEEP_DAYS. |
| `task setup` | Verify Go and Bun are installed. |
| `task completion:install` | Install task shell completion for the current shell. |
| `task migrate:create NAME=...` | Create a new PocketBase migration (use `snake_case`; name becomes part of the filename). |
| `task go:toolchain:update` | Update module toolchain to latest Go (Go 1.21+). |
| `task lint` | Run backend + frontend lint. |
| `task lint:backend` | Run backend lint. |
| `task lint:backend:fix` | Run backend lint with auto-fix (local only). |
| `task lint:frontend` | Run frontend lint. |
| `task lint:frontend:fix` | Run frontend lint with auto-fix (local only). |
| `task docker:build` | Build Docker image with version tags. |
| `task docker:up` | Start production compose. |
| `task docker:up:migrate` | Run migrations, then start production compose. |
| `task docker:dev:up` | Start dev compose (Vite + backend). |
| `task docker:dev:up:migrate` | Run migrations, then start dev compose. |
| `task docker:dev:down` | Stop dev compose. |
| `task docker:dev:logs` | Tail dev compose logs. |
| `task docker:dev:status` | Show dev compose status. |

## Notes
- Uploader binaries are downloaded from the **Uploader** screen and use the server’s base URL + token.
- Intel channel configuration is pulled by the uploader from the backend; no local XML config.
- Scheduled refresh tasks keep affiliations/scopes up to date and audit these runs.
