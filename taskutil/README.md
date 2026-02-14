# taskutil

`taskutil` is a small helper binary used by this repository's Task workflows.

## What it does

- Runs local Dev Console (`dev`) for frontend/backend processes.
- Derives build version (`version`) from git state.
- Prepares embedded frontend assets (`prepare-embed`).
- Performs repository cleanup (`clean-root`) using include/ignore rules.
- Tails and cleans dev logs (`dev-logs-tail`, `dev-logs-clean`).

## Configuration

`taskutil` loads `.env.taskutil` from repo root, then reads env vars and flags.

Common env vars:

- Project/root discovery:
  - `TASKUTIL_ROOT`: explicit project root path (skips marker discovery).
  - `TASKUTIL_ROOT_SIGNATURE`: comma-separated files/directories that must all exist for a directory to be treated as repo root during auto-discovery.
- Naming:
  - `TASKUTIL_APP_NAME`: app name used in UI titles and status text.
- Paths:
  - `TASKUTIL_FRONTEND_DIR`: frontend working directory.
  - `TASKUTIL_BACKEND_DIR`: backend working directory.
  - `TASKUTIL_BIN_DIR`: directory containing built binaries.
  - `TASKUTIL_BACKEND_BIN_NAME`: backend binary filename (for example `sentinel2-server`).
  - `TASKUTIL_BACKEND_BIN_PATH`: full override path to backend binary.
  - `TASKUTIL_EMBED_SRC`: frontend build output path copied for embed.
  - `TASKUTIL_EMBED_DEST`: backend embed destination path.
- Logs:
  - `LOG_DIR`: directory where dev session logs are written.
  - `LOG_JSON_PATH`: backend JSONL log file path.
- Dev console behavior:
  - `DEV_MIGRATIONS`: migrate command used by Dev Console action.
  - `EXPERIMENTAL_PTY`: enable experimental PTY process mode.
- Log utilities:
  - `TAIL_LINES`: default line count for `dev-logs-tail`.
  - `KEEP_DAYS`: retention window for `dev-logs-clean`.
- Cleanup behavior:
  - `TASKUTIL_CLEAN_RULES_FILE`: override path for cleanup rules file.
  - `TASKUTIL_CLEAN_YES`: skip confirmation prompt when set (non-interactive clean).

## Dev Console keybinds

- `q` / `Ctrl+C`: quit
- `tab`, `left/right`, `1`, `2`: pane focus
- `r`: restart focused process
- `f` / `b`: restart frontend/backend
- `Ctrl+F`: rebuild frontend and restart frontend
- `Ctrl+R`: rebuild backend and restart backend
- `Ctrl+G`: run migrate then restart backend
- `up/down/pgup/pgdn`: scroll focused pane
- `home` / `end`: jump to top / follow bottom
- `s`: switch layout (horizontal/vertical)
- `x`: swap pane process positions
- `z`: fullscreen focused pane
- `m`: toggle mouse capture
- `v`: toggle log-line-selection mode
- `c`: clear focused pane logs
- `h`: show/hide shortcut help
- mouse click+drag (non-fullscreen): drag-select text in the focused pane; selection copies to clipboard on release

## Cleanup rules (`.cleanrules`)

- Include: plain pattern (example: `dist/**`)
- Ignore: `!pattern` (example: `!dist/keep/**`)
- Ignore rules always win.
- Leading `/` anchors a rule to repo root (example: `/dist/`).
- Rule separators: comma, newline, or semicolon.
- Comments: `# ...` and inline `pattern # ...`.
- Escape literal leading control chars:
  - `\!file`
  - `\#file`

By default, rules are loaded from `./.cleanrules` when present, otherwise builtin defaults are used.
Set `TASKUTIL_CLEAN_RULES_FILE` to point to an alternate rules file.

`clean-root` shows a deletion plan (minimized targets + stats) and asks `y/N` unless `TASKUTIL_CLEAN_YES=1` (or `--yes`).
