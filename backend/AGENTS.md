# Backend AGENTS

## Purpose
This subtree owns the Sentinel 2 Go backend: HTTP handlers, auth providers, ESI clients, domain services, background jobs, middleware, server wiring, and PocketBase migrations.

## Ownership
- `backend/internal/`
- `backend/pb_migrations/`
- `backend/main.go`

## Local Contracts
- Follow the root AGENTS rules and the backend architecture guide in [`docs/BACKEND.md`](/home/terminal/Code/sentinel2/docs/BACKEND.md).
- Keep handlers thin and move domain behavior into the closest backend package.
- Preserve package boundaries such as `internal/auth`, `internal/esi`, `internal/intel`, `internal/jobs`, `internal/middleware`, `internal/server`, `internal/store`, and related domain packages.
- Use `task` from the repository root for backend workflows, especially `task lint:backend`, `task test:backend`, `task audit:dependencies`, `task build`, and `task build:migrate`.
- Prefer package constants and package errors for repeated values and shared failures.
- Keep migrations additive and aligned with PocketBase collection semantics.

## Work Guidance
- Use `mcp-gopls` for symbol lookup, references, renames, diagnostics, and Go test/coverage/vulnerability checks.
- Confirm method availability before introducing new calls or helper methods.
- Keep Go files formatted and grouped according to the project style rules.
- Update [`docs/BACKEND.md`](/home/terminal/Code/sentinel2/docs/BACKEND.md) when backend behavior, workflow, or ownership changes.

## Verification
- `task lint:backend`
- `task test:backend`
- `task build`
- `task build:migrate`
- `task audit:dependencies`

## Child DOX Index
- `backend/internal/`: Backend domain code and tests. No child `AGENTS.md` files currently exist under this subtree.
- `backend/pb_migrations/`: PocketBase migrations. No child `AGENTS.md` files currently exist under this subtree.
