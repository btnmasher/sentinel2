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
