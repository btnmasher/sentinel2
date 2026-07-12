# AGENTS Ruleset

This document is an execution policy for coding agents and contributors.

## Priority Order

1. Follow explicit user instructions for the current task.
2. Follow this `AGENTS.md` policy.
3. Preserve existing project conventions in nearby code.
4. Optimize for correctness, maintainability, and minimal-risk changes.

## Command & Workflow Policy

### Task-First Execution (MUST)

- Use `task` as the primary command interface.
- Prefer `task` over ad-hoc `go`, `bun`, or `docker` commands.
- If a repeated workflow is missing from `Taskfile.yml`, add a task.

### Standard Commands

- Setup: `task setup`
- Local dev: `task dev`
- Build: `task build`
- Build + migrations: `task build:migrate`
- Backend checks: `task lint:backend`
- Frontend checks: `task lint:frontend` (includes `bun run typecheck`)

## Safety & Change Hygiene

- Never commit secrets or credentials.
- Keep diffs minimal and directly related to the task.
- Update docs and task commands when operational behavior changes.

## Go Navigation & Refactoring Rules (mcp-gopls)

Always prioritize `mcp-gopls` tools for Go analysis.

- Searching symbols: use `search_workspace_symbols`.
- Finding references: use `find_references`.
- Definition lookup: use `go_to_definition`.
- Renaming symbols: use `rename_symbol`.
- Inspection: use `get_hover_info` and `check_diagnostics`.
- Module cleanup: use `run_go_mod_tidy` when imports/modules change.
- Test/quality/security:
  - `run_go_test`
  - `analyze_coverage`
  - `run_govulncheck`

### Verification & Deprecation Rules (MUST)

- Before suggesting creation of a method, verify existence with `get_hover_info`.
- Scan hover docs for `Deprecated:`.
- If deprecated:
  - Do not suggest the deprecated symbol by default.
  - Surface the recommended alternative when available.
- If user explicitly asks for deprecated usage:
  - Warn first and explain tradeoffs.
- If hover is ambiguous, cross-check deprecation with `check_diagnostics` (SA1019).

### Reasoning Loop (MUST)

1. If user asks to use method X, run `get_hover_info` on the receiver type.
2. If missing/symbol-not-found, only then suggest creating it.
3. If present, use the real signature from tool output.

### Fail-Safe

- Never assume method availability from naming conventions.
- Always confirm with LSP context first.

## Go Style Rules

1. Group each assignment + error check together, then separate groups with a blank line.
2. Do not shadow error variables in the same function scope.
3. Prefer early returns to reduce nesting.
4. Prefer modular package boundaries by domain (for example `internal/esi`, `internal/intel`, `internal/sde`, `internal/map`, `internal/auth`).
5. Use repository-style interfaces for persistence access.
6. Use constants for repeated strings; place package constants in `constants.go`.
7. Create `errors.go` per package for package-level errors; wrap with `%w` for `errors.Is/As`.
8. Use const groups for enum-like parameters with exported, prefixed names (for example `AuthProviderEVE`, `AuthProviderSSO`).
9. File structure order:
   1. Package-level vars
   2. Types
   3. Constructors
   4. Methods
   5. Exported functions
   6. Unexported functions

## GoDoc Rules

- Every exported identifier (type, function, method, const group, var) must have a GoDoc comment.
- The comment must start with the identifier name.
- Comments must be full sentences and explain behavior.
- Include non-obvious side effects when relevant.
- Do not restate the name without adding meaning.
- Write for readers unfamiliar with the local implementation.

## React + TypeScript Architecture (Feature-Based)

Use this structure for new code and refactors.

### Preferred Structure

```text
src/
├── main.tsx
├── app/
│   ├── App.tsx
│   ├── routes.tsx
│   └── store.ts
├── assets/
├── components/
├── config/
├── features/
│   └── [name]/
│       ├── api/
│       ├── components/
│       ├── types/
│       └── index.ts
├── hooks/
├── layouts/
├── pages/
├── types/
└── utils/
```

### Core Frontend Rules

- Encapsulation: keep features self-contained.
- Public API: import features via `features/[name]/index.ts` only; no deep imports.
- Page responsibility:
  1. Parse route params.
  2. Compose layout.
  3. Pass props into feature components.
- Proximity:
  - Feature-only component -> `features/[name]/components`
  - Shared across 2+ features -> `src/components`
- TypeScript placement:
  - Local types in component file.
  - Shared feature types in `features/[name]/types/index.ts`.
  - Global shared types in `src/types`.

### Page Glue Example

```tsx
// src/pages/ProjectDetails.tsx
import { useParams } from "react-router-dom";
import { MainLayout } from "@/layouts";
import { ProjectHeader, ProjectTaskBoard } from "@/features/projects";

export const ProjectDetailsPage = () => {
  const { id } = useParams<{ id: string }>();

  return (
    <MainLayout>
      <ProjectHeader projectId={id} />
      <ProjectTaskBoard projectId={id} />
    </MainLayout>
  );
};
```

### `tsconfig` Path Mapping

```json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```


# DOX framework

- DOX is highly performant AGENTS.md hierarchy installed here
- Agent must follow DOX instructions across any edits

## Core Contract

- AGENTS.md files are binding work contracts for their subtrees
- Work products, source materials, instructions, records, assets, and durable docs must stay understandable from the nearest applicable AGENTS.md plus every parent AGENTS.md above it

## Read Before Editing

1. Read the root AGENTS.md
2. Identify every file or folder you expect to touch
3. Walk from the repository root to each target path
4. Read every AGENTS.md found along each route
5. If a parent AGENTS.md lists a child AGENTS.md whose scope contains the path, read that child and continue from there
6. Use the nearest AGENTS.md as the local contract and parent docs for repo-wide rules
7. If docs conflict, the closer doc controls local work details, but no child doc may weaken DOX

Do not rely on memory. Re-read the applicable DOX chain in the current session before editing.

## Update After Editing

Every meaningful change requires a DOX pass before the task is done.

Update the closest owning AGENTS.md when a change affects:

- purpose, scope, ownership, or responsibilities
- durable structure, contracts, workflows, or operating rules
- required inputs, outputs, permissions, constraints, side effects, or artifacts
- user preferences about behavior, communication, process, organization, or quality
- AGENTS.md creation, deletion, move, rename, or index contents

Update parent docs when parent-level structure, ownership, workflow, or child index changes. Update child docs when parent changes alter local rules. Remove stale or contradictory text immediately. Small edits that do not change behavior or contracts may leave docs unchanged, but the DOX pass still must happen.

## Hierarchy

- Root AGENTS.md is the DOX rail: project-wide instructions, global preferences, durable workflow rules, and the top-level Child DOX Index
- Child AGENTS.md files own domain-specific instructions and their own Child DOX Index
- Each parent explains what its direct children cover and what stays owned by the parent
- The closer a doc is to the work, the more specific and practical it must be

## Child Doc Shape

- Create a child AGENTS.md when a folder becomes a durable boundary with its own purpose, rules, responsibilities, workflow, materials, or quality standards
- Work Guidance must reflect the current standards of the project or user instructions; if there are no specific standards or instructions yet, leave it empty
- Verification must reflect an existing check; if no verification framework exists yet, leave it empty and update it when one exists

Default section order:
- Purpose
- Ownership
- Local Contracts
- Work Guidance
- Verification
- Child DOX Index

## Style

- Keep docs concise, current, and operational
- Document stable contracts, not diary entries
- Put broad rules in parent docs and concrete details in child docs
- Prefer direct bullets with explicit names
- Do not duplicate rules across many files unless each scope needs a local version
- Delete stale notes instead of explaining history
- Trim obvious statements, repeated rules, misplaced detail, and warnings for risks that no longer exist

## Closeout

1. Re-check changed paths against the DOX chain
2. Update nearest owning docs and any affected parents or children
3. Refresh every affected Child DOX Index
4. Remove stale or contradictory text
5. Run existing verification when relevant
6. Report any docs intentionally left unchanged and why

## User Preferences

When the user requests a durable behavior change, record it here or in the relevant child AGENTS.md

## Child DOX Index

- `backend/`: Go backend services, API handlers, auth providers, migrations, and backend tests. Child contract: [`backend/AGENTS.md`](/home/terminal/Code/sentinel2/backend/AGENTS.md)
- `frontend/`: React frontend, feature modules, shared UI, route pages, and frontend tests. Child contract: [`frontend/AGENTS.md`](/home/terminal/Code/sentinel2/frontend/AGENTS.md)
- `uploader/`: Standalone uploader app, runtime, UI, networking, and release tooling. Child contract: [`uploader/AGENTS.md`](/home/terminal/Code/sentinel2/uploader/AGENTS.md)
- `taskutil/`: Repository task helper binary, dev console, cleanup tooling, and log utilities. Child contract: [`taskutil/AGENTS.md`](/home/terminal/Code/sentinel2/taskutil/AGENTS.md)
