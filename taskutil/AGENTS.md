# taskutil AGENTS

## Purpose
This subtree owns the task helper binary used by the repository's root Task workflows, including the dev console, cleanup tooling, build metadata, and log helpers.

## Ownership
- `taskutil/main.go`
- `taskutil/internal/`
- `taskutil/README.md`

## Local Contracts
- Follow the root AGENTS rules for Go code and the taskutil usage guide in [`taskutil/README.md`](/home/terminal/Code/sentinel2/taskutil/README.md).
- Treat `taskutil` as a repo toolchain component, not a standalone product boundary.
- Keep behavior aligned with the root `Taskfile.yml` workflows that invoke taskutil binaries and helper commands.
- Use taskutil's own code paths for dev console, cleanup, project discovery, versioning, and log utilities instead of duplicating that logic elsewhere.

## Work Guidance
- Keep helper behavior deterministic and conservative because it is invoked by repository-wide workflows.
- Update [`taskutil/README.md`](/home/terminal/Code/sentinel2/taskutil/README.md) when environment variables, keybinds, or taskutil responsibilities change.
- Preserve OS-specific split files and keep platform handling explicit.

## Verification
- `task build`
- `task dev`
- `task dev:logs`
- `task dev:logs:clean`

## Child DOX Index
- `taskutil/internal/`: Task helper internals for dev console, cleanup, build metadata, and log tooling. No child `AGENTS.md` files currently exist under this subtree.
