# Go Backend Architecture

This document summarizes how the Sentinel 2 backend is organized and how to extend it safely. Use it as a guide when adding API endpoints, services, or background jobs.

## 1. Directory Structure
```text
backend/
├── main.go                  # Entry point (config, start server)
├── internal/
│   ├── api/                 # HTTP handlers by domain
│   │   ├── admin/           # Admin APIs + map data updates
│   │   ├── auth/            # Login/link endpoints + callbacks
│   │   ├── intel/           # Intel ingestion + subscriptions
│   │   ├── maps/            # Map routes, search, character locations
│   │   └── staff/           # Staff tools (channels, jumpbridges)
│   ├── auth/                # Auth providers, refreshers, session utilities
│   ├── cleanup/             # Cleanup jobs/services
│   ├── config/              # Config parsing + defaults
│   ├── esi/                 # ESI clients (direct/proxy)
│   ├── intel/               # Intel domain logic (routing, top routes)
│   ├── jobs/                # Background job runners + constants
│   ├── logging/             # Structured logging helpers
│   ├── middleware/          # Auth, staff/admin gating, rate limiting
│   ├── oidc/                # OIDC client integration
│   ├── server/              # Server wiring: routes, cron, dependencies
│   │   ├── server.go        # Dependency graph + app startup
│   │   ├── routes.go        # HTTP routes + middleware bindings
│   │   ├── cron.go          # Cron registration/orchestration
│   │   └── cron_*.go        # Job specs and run helpers
│   ├── store/               # PocketBase collection names + helpers
│   ├── utils/               # Shared utilities
│   └── web/                 # Embedded frontend + proxy handling
├── pb_migrations/           # PocketBase Go migrations
```

## 2. Core Development Rules
- **Domain isolation:** Keep feature logic in the closest domain package (`internal/intel`, `internal/auth`, `internal/esi`), not in handlers.
- **Handlers are thin:** API handlers should parse/validate inputs and call domain services.
- **Single entrypoint:** The server wiring lives in `internal/server` and should be the only place wiring dependencies.
- **Auth gating:** Enforce access control via middleware in `internal/middleware` and route groups in `internal/server/routes.go`.
- **Prefer services over globals:** Long-running or reusable logic should be in a service struct rather than package-level functions.
- **PocketBase first:** Data access should use PocketBase collections with types/IDs defined in `internal/store`.

## 3. OAuth Security Contract
- Production OAuth callbacks use the configured `PUBLIC_BASE_URL`, which must be an HTTPS origin. Request `Host`, `Origin`, and `Referer` values do not override it.
- Local development may omit `PUBLIC_BASE_URL`; callback origins are then accepted only from loopback hosts and may use HTTP for the local Vite setup.
- OAuth access, refresh, expiry, and scope fields remain server-side PocketBase data and are hidden from collection API serialization. Auth exchange and refresh responses use an explicit browser-safe DTO.
- Run `task audit:dependencies` before release. The Go scan reports reachable vulnerabilities separately from advisories in unused packages within required modules.

## 4. Handler → Service Flow
The typical request flow is: route handler → input validation → domain/service call → structured response. Keep handlers thin and move heavy logic into services or domain packages.

### Examples
- Map handler example:
```go
func (h *MapHandler) regions(c *core.RequestEvent, mode string) error {
  // Parse + validate inputs from the request.
  regionIDs, parseErr := h.parseRegionIDs(c.Request.PathValue("regions"))
  if parseErr != nil {
    return router.NewBadRequestError("Invalid region list.", logging.Fields{
      "regions": c.Request.PathValue("regions"),
      "mode":    mode,
    })
  }

  // Delegate to domain helpers/services.
  systems, systemsErr := h.fetchSystems(regionIDs, mode)
  if systemsErr != nil {
    return router.NewInternalServerError("Failed to load systems.", logging.Fields{
      "region_ids": regionIDs,
      "mode":       mode,
    })
  }

  gates, gatesErr := h.fetchGates(regionIDs)
  if gatesErr != nil {
    return router.NewInternalServerError("Failed to load stargates.", logging.Fields{
      "region_ids": regionIDs,
    })
  }

  regions, regionsErr := h.fetchRegions(regionIDs, mode)
  if regionsErr != nil {
    return router.NewInternalServerError("Failed to load regions.", logging.Fields{
      "region_ids": regionIDs,
    })
  }

  normalizeSystemsByRegion(systems, regionIDs, 1000, 1000)
  normalizeRegions(regions)
  jumpbridges, jumpbridgesErr := h.fetchJumpbridges(regionIDs)
  if jumpbridgesErr != nil {
    return router.NewInternalServerError("Failed to load jumpbridges.", logging.Fields{
      "region_ids": regionIDs,
    })
  }

  // Return a structured response.
  return c.JSON(http.StatusOK, MapResponse{
    Regions:     regions,
    Systems:     systems,
    Gates:       gates,
    Jumpbridges: jumpbridges,
  })
}
```

## 5. Job Runner
Sentinel uses a structured job runner to track background work in PocketBase and emit consistent logs.

### Core Concepts
Core concepts:
- **Job records + steps:** `Runner.Run` creates a job record; `stepper.Run` creates step records under the same job ID.
- **Partial + skipped states:** `stepper.SkipParent(reason)` marks the job skipped. `stepper.Partial(err)` marks a recoverable issue, but final status is computed from step outcomes.
- **Timeouts + cancellation:** `Timeout` controls max runtime; cancellation is recorded as a canceled job.
- **Uniqueness:** `Unique: true` prevents overlapping runs for the same `Kind` + `Step`.
- **Metadata:** `Kind`, `Step`, `Trigger`, and optional `ActorID` are persisted and logged.

### API Overview
API overview (when to use what):
- `jobs.NewRunner(app, RunOptions)` creates a runner and job record tracker.
- `RunOptions.JobName` sets the display name; if empty, `Kind` is used.
- `RunOptions.JobOptions` sets `Kind`, `Step`, `Trigger`, `ActorID`, and `Hidden`.
- `RunOptions.Timeout` defaults to 10 minutes; set `jobs.NoTimeout` to disable.
- `RunOptions.Unique` skips if another job of the same `Kind` + `Step` is running.
- `RunOptions.JobFunc` allows injecting context metadata (e.g., auth info).
- `runner.WithFields(fields)` adds structured log fields for the job record.
- `runner.WithMessage(message)` adds a message to the completion log entry.

### Stepper API
Stepper API:
- `stepper.Run(name, critical, fn)` runs a step and records its status.
- `stepper.Partial(err)` records a recoverable issue on the parent job.
- `stepper.SkipStep(name, reason)` records a skipped step without failing the job.
- `stepper.SkipParent(reason)` skips the entire job (returns `ErrJobSkipped`).

### Logging
Logging behavior:
- Each job and step emits `started` and `completed` logs with `duration_ms` and `status`.
- Status values: `success`, `failed`, `partial`, `skipped`, `canceled`, `timeout`.
- Logs include `job_id`, `kind`, and `step`, plus `trigger` and `actor_id` if present.
- Message text is explicit by outcome (for example: `job completed`, `job completed with errors`, `job failed`, `job timed out`).

Finalization rules:
- Any critical step failure fails the parent job.
- Non-critical failures can produce `partial` only when at least one step succeeded and no critical steps failed.
- Skipped steps are not failures.
- If a parent has recoverable errors but no qualifying successful step, it finalizes as `failed`.

### Usage
Typical usage:
```go
runner := jobs.NewRunner(app, jobs.RunOptions{
  JobName: "admin.character_refresh",
  JobOptions: jobs.JobOptions{
    Kind:    "character_refresh",
    Trigger: "admin.character_refresh",
    ActorID: actorID,
  },
  Timeout: 5 * time.Minute,
  Unique:  true,
  JobFunc: func(ctx context.Context) context.Context {
    return auth.WithRefreshJobMeta(ctx, "admin.character_refresh", actorID)
  },
})

_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
  if err := stepper.Run("load_records", true, func(ctx context.Context) error {
    // load input
    return nil
  }); err != nil {
    return err
  }

  if err := stepper.Run("refresh_batch", false, func(ctx context.Context) error {
    // do work; mark partial on non‑fatal errors
    return nil
  }); err != nil {
    return err
  }

  return nil
})
```

Real example (admin-triggered map data update):
```go
func (h *MapUpdateHandler) RunAll(c *core.RequestEvent) error {
  actorID := ""
  if c.Auth != nil {
    actorID = c.Auth.Id
  }
  // Create the job and return immediately (async job).
  runner := jobs.NewRunner(h.App, jobs.RunOptions{
    JobName: mapdata.JobMapDataUpdate,
    JobOptions: jobs.JobOptions{
      Kind:    "map_data_update",
      Trigger: jobs.TriggerAdminManual,
      ActorID: actorID,
    },
    Timeout: jobs.NoTimeout,
  })
  jobID := runner.JobID()
  logging.WithRequest(h.App, c).
    WithFields(logging.Fields{
      "job_id": jobID,
      "cron":   jobs.TriggerAdminManual,
      "force":  true,
    }).
    Info("map data update requested")

  go func(jobID string, actorID string) {
    baseCtx, cancel := context.WithCancel(context.Background())
    defer cancel()
    localRunner := jobs.NewRunner(h.App, jobs.RunOptions{
      JobID: jobID,
      JobOptions: jobs.JobOptions{
        Kind:    "map_data_update",
        Trigger: jobs.TriggerAdminManual,
        ActorID: actorID,
      },
      Timeout: jobs.NoTimeout,
      Parent:  baseCtx,
    })
    // Run the job work in a separate goroutine.
    mapdata.RunMapDataUpdateWithContext(baseCtx, h.App, localRunner, jobs.TriggerAdminManual, true)
  }(jobID, actorID)

  return c.JSON(http.StatusAccepted, mapDataResponse{
    JobID: jobID,
    Step:  "all",
  })
}
```

### Cancellation
Cancellation notes:
- Jobs respect context cancellation (`context.Canceled`) and are recorded as canceled.
- Timeouts result in `context.DeadlineExceeded` and a `timeout` status.
- Use `RunOptions.Parent` to bind a job to a caller’s context when needed.

### Where to Use
Where to use:
- **Cron jobs:** `internal/server/cron.go`, `internal/server/cron_*.go`
- **Admin‑triggered jobs:** `internal/api/admin/handler.go`
- **Background refreshers:** `internal/auth/character_refresh.go`

## 6. Logging
Logging is centralized via `internal/logging` and uses structured fields for filtering in PocketBase logs.
Set `DEBUG_ENABLED=true` (or run the server with `--dev`) to enable verbose debug logging, including SQL statements.
Optional logging controls:
- `LOG_LEVEL=debug|info|warn|error`
- `LOG_PRETTY=true` (prints a styled console view for app logs)
- `LOG_PRETTY_PB=true` (experimental: replaces PocketBase dev console printer using go:linkname)
- `LOG_JSON=true` (write JSONL logs for app output)
- `LOG_JSON_PATH=/path/to/backend.jsonl`
- `LOG_JSON_PB=true` (experimental: also write PocketBase dev console logs to JSONL)
- `INTEL_REPORT_HASH_SLOTS=20` (dedupe slots per report fingerprint)

### Guidelines
Guidelines:
- Use `logging.WithRequest(app, c)` inside request handlers to include request context.
- Prefer `WithFields(logging.Fields{...})` for structured metadata like IDs, counts, and durations.
- Use `Info` for expected state changes, `Warn` for recoverable issues, and `Error` for failures.

### Common Fields
Common fields:
- `request_id`, `type`, `status`, `duration_ms`
- Domain-specific IDs like `user_id`, `character_id`, `job_id`, `region_ids`

### Example
Example:
```go
logging.WithRequest(h.App, c).
  WithFields(logging.Fields{
    "user_id":   user.Id,
    "system_id": systemID,
  }).
  Info("map system selected")
```

## 7. PocketBase Primer
PocketBase is the application core for storage, auth, and request/response handling.

### Router + Handlers
- Handlers receive `*core.RequestEvent` and use the PocketBase `router` helpers.
- Use `c.Request` for path params/body and `c.JSON(status, payload)` for responses.
- Use `router.NewBadRequestError`, `router.NewNotFoundError`, `router.NewInternalServerError` for consistent API errors.
- Auth is attached to `c.Auth`; helpers like `auth.CurrentUser(c)` centralize access checks.

### Records + Collections
- Use `app.FindRecordById` / `app.FindRecordsByFilter` to fetch records.
- Use `record.GetString`, `record.GetInt`, `record.GetBool` to read typed fields safely.
- Use `record.Set` + `app.Save(record)` to persist mutations.
- Collection names and helpers live in `internal/store` to avoid magic strings.

### Migrations
- PocketBase migrations are Go files in `backend/pb_migrations`.
- They are compiled into the server binary; rebuild after any migration changes.
- The build scripts include `--migrate`/`--migrate-run` flags to apply them.

### Further Reading
- [PocketBase Go docs](https://pocketbase.io/docs/go)
- [PocketBase records](https://pocketbase.io/docs/go-records)
- [PocketBase router + events](https://pocketbase.io/docs/go-event-hooks)
- [PocketBase filters](https://pocketbase.io/docs/go-records/#query-filters)

## 8. Linting (Go)
We use `golangci-lint` for backend linting.

Install:
- Recommended (binary install): `curl -sSfL https://golangci-lint.run/install.sh | sh -s v2.9.0`
- macOS (brew): `brew install golangci-lint`
- Windows: `choco install golangci-lint` or `scoop install main/golangci-lint`
- Go install (not recommended by upstream): `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
Note: upstream recommends binary installs and discourages `go install` for golangci-lint.

Run via build script:
```bash
task lint:backend
```

Auto-fix (local only):
```bash
task lint:backend:fix
```

Run directly:
```bash
golangci-lint run ./...
```

### Automated Checks
1. Error variable shadowing is disallowed (`govet` with `check-shadowing`).
2. Complex nesting and high cognitive complexity should be avoided (`nestif`, `gocognit`).
3. Error wrapping/handling correctness (`errorlint`, `errcheck`).
4. Misuse of contexts and loop variables (`noctx`, `copyloopvar`).
5. Unused or suspicious code (`staticcheck`, `ineffassign`, `unparam`, `unconvert`, `wastedassign`).
6. Spelling and lint suppression hygiene (`misspell`, `nolintlint`).

### Manual Checks
1. Group assignment and error checks together, then add a blank line before the next group.
2. Prefer early returns and avoid deep nesting by flattening control flow.
3. Prefer package boundaries by domain rather than a monolithic services package.
4. Use repository or interface boundaries for persistence and external integrations.
5. Centralize repeated string literals in `constants.go` per package.
6. Define sentinel errors in `errors.go` per package using `errors.Is/As`.
7. Use exported const groups as enums with a shared prefix.
8. Order file contents: package vars, types, constructors, methods, exported funcs, unexported funcs.
9. Keep comments meaningful and user-focused when added; avoid noise.
