# Sentinel 2 Frontend Architecture

This document describes the Sentinel 2 frontend structure, conventions, and how to keep the codebase consistent as it grows.

## 1. Directory Structure
```text
src/
├── main.tsx         # App bootstrap
├── app/             # Global setup (stores, config wiring)
├── assets/          # Static files and global CSS
├── components/      # Shared UI building blocks (Toast, AccordionCard, etc.)
├── config/          # API clients + runtime config (api.ts, pb.ts)
├── features/        # Domain modules (intel, map, navigation, staff, admin, etc.)
├── hooks/           # Reusable hooks (useEveImage, etc.)
├── layouts/         # Page shells (MainLayout)
├── pages/           # Route-level glue components
├── types/           # App-wide shared types
├── utils/           # Pure helpers
└── styles.css       # Tailwind additions + app-wide styles
```

Feature layout convention (example):
```text
features/map/
├── components/      # UI and rendering (MapCanvas, ContextMenu, MapSystem)
├── store/           # Zustand store + helpers
├── types/           # Map-specific types
└── index.ts         # Feature public API (export boundary)
```

## 2. Core Development Rules
- **Encapsulation:** Features should be self-contained as possible.
- **Public API:** Only import from a feature via its `index.ts`. Never deep-import (e.g., NO `import { X } from '@/features/auth/components/Button'`).
- **Page responsibility:** Pages parse URL params, wrap features in a layout, and render the feature entry point.
- **Proximity:** If a component is used in only one feature, keep it in `features/[name]/components`. If used in 2+ features, move it to `src/components`.
- **TypeScript:** 
    - Define local types in the component file.
    - Define shared feature types in `features/[name]/types/index.ts`.
    - Define global API/Utility types in `src/types/`.

## 3. Page Glue Example
```tsx
// src/pages/IntelPage.tsx
import MainLayout from "@/layouts/MainLayout";
import { Intel } from "@/features/intel";

export default function IntelPage() {
  return (
    <MainLayout>
      <Intel />
    </MainLayout>
  );
}
```

## 4. State + Data Flow
- **Zustand stores:** App-level stores live in `src/app/store` (auth, settings, UI, app config). Feature stores live under `features/[name]/store`.
- **API calls:** Prefer `src/config/api.ts` for REST calls; it attaches auth headers and handles common errors.
- **PocketBase client:** `src/config/pb.ts` configures the PB client and disables auto-cancellation to avoid dropped long polls.
- **Realtime:** Subscriptions live in feature hooks (e.g., `features/intel/hooks/useIntelRealtime.ts`).

## 5. Component Organization
- **Shared UI:** Put reusable UI in `src/components`.
- **Feature UI:** Keep domain-specific UI in `features/[name]/components`.
- **Map rendering:** `features/map/components` owns map rendering, context menus, and overlays.
- **Dialogs:** All app dialogs live in `src/components/dialogs` and are mounted in `MainLayout`.

## 6. Style Guide (Frontend)
- **Tailwind + daisyUI:** Use utility-first classes; prefer daisyUI tokens for consistent theming.
- **Custom styles:** Use `src/styles.css` for app-wide classes and overrides (map styles, context menu styles).
- **Typography:** Headings use `font-display` (Space Grotesk). Body text uses `font-body` (Inter).
- **UI feedback:** Use `Toast` for user-visible errors; include `meta` for request context when possible.

## 7. PocketBase Frontend Primer
PocketBase is the frontend’s realtime datastore and auth session source. We use the JS SDK for collection reads/writes and subscriptions, while custom API endpoints go through `src/config/api.ts`.

### When to use PocketBase vs REST
- **PocketBase SDK:** direct collection access (admin/staff tools, realtime feeds).
- **REST API (`api.ts`):** domain endpoints with server-side logic (maps, routing, auth flows).

### SDK Setup
```ts
// src/config/pb.ts
import PocketBase from "pocketbase";

export const pb = new PocketBase("/");
pb.autoCancellation(false);
```

### Auth State
```ts
// src/app/store/authStore.ts
pb.authStore.onChange(() => {
  // derive isAuthenticated, roles, and user ID
});
```

### Record Operations
```ts
// Read a collection
const records = await pb.collection("intel_channels").getFullList();

// Create
await pb.collection("intel_channels").create({ channel_name: "intel" });

// Update
await pb.collection("intel_channels").update(recordId, { channel_name: "new-name" });

// Delete
await pb.collection("intel_channels").delete(recordId);
```

### Realtime Subscriptions
```ts
// src/features/intel/hooks/useIntelRealtime.ts
const unsubscribe = await pb.collection("intel_reports").subscribe("*", (event) => {
  // handle insert/update/delete
});

// cleanup
unsubscribe();
```

### Error Handling Patterns
Use consistent toasts + logs for user-facing failures:
```ts
import { useUIStore } from "@/app/store/uiStore";

try {
  await pb.collection("intel_channels").create({ channel_name: "intel" });
} catch (error) {
  useUIStore.getState().setToast({
    text: "Failed to create intel channel",
    color: "error",
    meta: { error },
  });
}
```

Use the shared REST client for API endpoints (built-in auth headers + error handling):
```ts
import { api } from "@/config/api";

try {
  await api.post("/staff/jumpbridges/import", { jumpbridges: payload });
} catch (error) {
  useUIStore.getState().setToast({
    text: "Jumpbridge import failed",
    color: "error",
    meta: { error },
  });
}
```

### Record Permissions Notes
PocketBase collection rules must permit the operation:
- **Read:** list/view rules must allow the authenticated user.
- **Write:** create/update/delete rules must allow the operation (often staff/admin only).
- **Realtime:** subscription requires list/view access to the collection.

If an operation fails with `403`, confirm collection rules in `pb_migrations` or PocketBase admin UI.

### Usage Guidelines + Common Pitfalls
- Keep subscriptions in hooks with cleanup to avoid duplicate listeners and leaks.
- Use the REST `api.ts` client for server endpoints; use the PocketBase SDK for direct collection access.
- Ensure `authStore` reacts to `pb.authStore.onChange()` so token expiry doesn’t silently drop auth state.
- If you see `ClientResponseError: canceled`, confirm `pb.autoCancellation(false)` is set (we do this in `config/pb.ts`).
- `403` on collection calls usually means list/view/create/update/delete rules in `pb_migrations` are too strict.

### Further Reading
- [PocketBase JS SDK](https://pocketbase.io/docs/js-sdk)
- [PocketBase Realtime](https://pocketbase.io/docs/js-realtime)
- [PocketBase Auth Store](https://pocketbase.io/docs/js-authentication)

## 8. TypeScript Config (tsconfig.json)
Highlights:
- Builds for modern evergreen browsers with DOM typings.
- Bundler‑style module resolution aligned with Vite.
- `@/` alias for imports from `src/`.
- Type-check only (no emit) and safe for single‑file transforms.
- Uses the modern React JSX runtime.
- Strict type checking enabled.
- Skips third‑party type checks for faster builds.

## 9. Linting (Frontend)
Run via lint script:
```bash
task lint:frontend
```

Auto-fix (local only):
```bash
task lint:frontend:fix
```
