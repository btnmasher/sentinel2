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

- [docs/FRONTEND.md](docs/FRONTEND.md)
- [docs/BACKEND.md](docs/BACKEND.md)

### Development Workflow
`Task` is the task runner used for this repo's `task ...` commands.
- Project: https://taskfile.dev
- Install (Go): `go install github.com/go-task/task/v3/cmd/task@latest`
- Other install methods are listed in the Task docs: https://taskfile.dev/installation/


Task-based development, build, Docker, and CI/release command trees are documented in:
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)

Platform-specific setup for Windows/WSL is documented in:
- [docs/WINDOWS_DEVELOPMENT.md](docs/WINDOWS_DEVELOPMENT.md)

## Key Capabilities
- Realtime intel feed with filters and alarms.
- Region map with routes, waypoints, jump bridge links, and multiple layouts.
- Dedicated navigation page for quick route planning.
- Uploader companion tool with per-account tokens and cross-platform downloadable bundles.
- Admin/staff views for moderation, account management, and audits.

### Notes.
- Intel channel configuration is pulled by the uploader application from the backend; no need manually to maintain a local configuraton file.

## Auth Modes
- **EVE OAuth (standalone):** Character linking, profile management, admin tools.
- **TEST Auth (OIDC):** Uses OIDC claims and roles; account/character management is handled externally (no in-app character linking).

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

## Logging
The backend uses PocketBase's built-in `slog` logger, so logs show up in the PocketBase Logs UI. Request logs include a `request_id` plus `type`, `status`, and `duration_ms` fields. Domain logs (auth, intel, map, SDE, cleanup) use structured fields like `type` to make filtering easier.
View logs in the PocketBase Admin UI under **Logs**.
Optional verbose debugging (prints SQL + debug logs):
- `DEBUG_ENABLED=true` (or pass `--dev` to the server)
Optional log settings:
- `LOG_LEVEL=debug|info|warn|error` (default: info)
- `LOG_PRETTY=true` (applies a visual style to console logs for enhanced human readability, default: false)
- `LOG_PRETTY_PB=true` (experimental: hijacks PocketBase internal logger to apply a consistent visual style to request/datbase logs, default: false)
- `LOG_JSON=true` (write sentinel application logs to JSONL, default: false)
- `LOG_JSON_PATH=/path/to/backend.jsonl`
- `LOG_JSON_PB=true` (experimental: hijacks PocketBase inernal request/database logger to also output those logs to JSONL, default: false)
- `INTEL_REPORT_HASH_SLOTS=20` (dedupe slots per report fingerprint, default: 20)

## Docker
The Docker build uses `task build` inside a dedicated builder stage, so Docker and local/CI builds share the same Taskfile workflow. The builder cross-compiles uploader binaries and embeds frontend assets into the backend binary.
For Docker-based workflows, build dependencies are provided inside the image/toolchain stages (Go, Bun, Task, Zig, zip). Locally, you only need Docker and Docker Compose.
```
docker build -t sentinel2 .
docker run -p 8090:8090 -v pb_data:/app/pb_data sentinel2
```

Or with compose:
```
docker compose up --build
```

Useful commands:
Task-wrapped commands (`task docker:*`) require local Task runner to be installed.
- `task docker:build` build the image locally with version tagging.
- `task docker:up` start production compose.
- `task docker:up:detach` start production compose in detached mode.
- `task docker:up:migrate` run migrations, then start production compose.
- `task docker:up:migrate:detach` run migrations, then start production compose in detached mode.
- `task docker:logs` tail production compose logs.
- `task docker:status` show production compose status.
- `task docker:down` stop production compose.
- `task docker:restart` alias for `task docker:up:detach`.
- `task docker:restart:migrate` alias for `task docker:up:migrate:detach`.
Raw Docker/Compose fallback (no Task runner required):
- `docker run -p 8090:8090 -v pb_data:/app/pb_data sentinel2` run with a named volume.
- `docker compose up --build` build and run via compose.
- `docker compose logs -f` tail compose logs.
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
