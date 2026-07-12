# Frontend AGENTS

## Purpose
This subtree owns the Sentinel 2 React frontend, including route pages, feature modules, shared UI, application stores, hooks, and styling.

## Ownership
- `frontend/src/`
- `frontend/public/`
- `frontend/index.html`
- `frontend/vite.config.ts`
- `frontend/eslint.config.js`

## Local Contracts
- Follow the root AGENTS rules and the frontend architecture guide in [`docs/FRONTEND.md`](/home/terminal/Code/sentinel2/docs/FRONTEND.md).
- Preserve the feature-based structure: route glue in `src/pages`, feature logic in `src/features`, shared components in `src/components`, and app-level state in `src/app`.
- Import feature code through each feature's `index.ts` public API; do not deep-import feature internals.
- Use the established modal and dialog patterns rather than ad hoc overlay state.
- Keep shared UI and feature-specific UI separated by reuse scope.
- Use `task` from the repository root for frontend workflows, especially `task lint:frontend`, `task build:frontend`, and `task dev:frontend`.

## Work Guidance
- Keep React changes aligned with the existing feature boundaries and composition style.
- Prefer local types near the component that owns them, and shared feature types in `features/[name]/types`.
- Update [`docs/FRONTEND.md`](/home/terminal/Code/sentinel2/docs/FRONTEND.md) when UI architecture, modal behavior, or feature boundaries change.

## Verification
- `task lint:frontend`
- `task build:frontend`
- `task dev:frontend`

## Child DOX Index
- `frontend/src/`: React application source. No child `AGENTS.md` files currently exist under this subtree.
