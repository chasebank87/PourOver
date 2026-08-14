# PourOver v1 Design

**Status:** Approved for planning  
**Date:** 2026-05-18  
**Scope:** macOS-only, CLI-first

## Vision

PourOver is a declarative macOS environment manager focused on Homebrew packages, dotfiles, and config files. It aims to provide "Nix-like outcomes" with lower complexity and native fit for macOS workflows.

Version 1 is intentionally CLI-first. Admin dashboard work is deferred to v2 so the core reconciliation engine, config model, and safety behaviors can be stabilized first.

## Product Goals

- Declarative desired state for brew packages and file management
- One command to reconcile system state (`pourover apply`)
- Safe defaults for destructive operations
- Plain-file state for portability, inspectability, and git-friendly backup
- Optional iCloud sync for backup/restore with automatic mirror on apply

## Non-Goals (v1)

- Native dashboard UI
- Remote multi-host orchestration
- Full Nix-like build/store semantics
- Non-macOS support

## Core User Experience

### Commands

- `pourover init`: scaffold baseline config and modules
- `pourover plan`: compute and print proposed actions
- `pourover apply`: execute reconciled actions
- `pourover doctor`: verify prerequisites and environment health
- `pourover backup`: force state snapshot backup
- `pourover restore`: restore latest or selected snapshot

### Safety behavior

- Default uninstall policy is **safe confirm** (prompt before uninstalling undeclared packages)
- Config can override policy to:
  - `strict` (auto-prune undeclared packages)
  - `non_destructive` (never uninstall unless explicit flag in future extension)

## Configuration Model

### Entry point and composition

- Root file: `pourover.lua`
- Supports Lua module imports for hybrid organization:
  - `require("packages")`
  - `require("dotfiles")`
  - `require("configs")`

### Declarative schema (conceptual)

- `packages.formulae`: brew formula list
- `packages.casks`: brew cask list
- `files.links`: source/target symlink declarations (v1)
- `files.managed`: reserved for v2 (copy/template); see `docs/v2-backlog.md`
- `policy.uninstall_mode`: `safe` | `strict` | `non_destructive`
- `backup.icloud.enabled`: boolean
- `backup.icloud.path`: optional custom iCloud path override

Go validates and normalizes the loaded Lua result into a strict internal manifest before planning/applying.

### Paths (config vs state vs managed targets)

PourOver separates three locations on purpose:

| Role | Default path | Notes |
|------|----------------|-------|
| **PourOver config** | `~/.pourover/` | Declarative Lua (`pourover.lua`, modules). Not under `~/.config` so it stays distinct from app configs PourOver manages. |
| **Runtime state** | `~/Library/Application Support/PourOver/state/` | Lock, history, snapshots — machine-local, not usually git-tracked. |
| **Managed targets** | `~/.config/<app>/`, `~/.zshrc`, etc. | Declared in Lua; `apply` reconciles these on disk. |

Config discovery:

- **Default:** load `~/.pourover/pourover.lua` (and resolve `require()` from `~/.pourover/`).
- **Override:** `--config /path/to/pourover.lua` (module search path = directory containing that file).

Users may keep a separate dotfiles repo (e.g. `~/dotfiles/`); PourOver does not assume or require that layout. They can symlink or copy into `~/.pourover/` if they want git-backed config elsewhere.

## Architecture

### Runtime split

- **Go core**
  - CLI commands and UX
  - Manifest validation and normalization
  - State discovery (brew/filesystem)
  - Diff/reconciliation engine
  - Executor and safety prompts
  - Plain-file state and backup management
- **Lua layer** (`github.com/yuin/gopher-lua`, MIT)
  - User-authored declaration format
  - Composition primitives via modules/imports

### Major components

- `internal/config`: Lua loading, schema validation, manifest normalization
- `internal/discovery`: current-state readers (brew packages, file links/status)
- `internal/plan`: desired-vs-current diff and action graph generation
- `internal/exec`: action executor and prompt flow
- `internal/state`: lock/history snapshots, integrity hashes, backup/restore
- `internal/backup`: iCloud target resolution and file sync routines

## Data Flow

### `plan`

1. Load `pourover.lua` and imported modules
2. Normalize to internal manifest
3. Discover actual system state
4. Compute diff and ordered action plan
5. Render deterministic output (human-readable + optional JSON)

### `apply`

1. Re-run planning pipeline for current truth
2. Evaluate uninstall actions against policy
3. If policy is `safe`, prompt for confirmation once per run
4. Execute actions in stable order:
   - taps
   - formula installs/removals
   - cask installs/removals
   - file link/copy/unlink operations
5. Persist state/history snapshots
6. Mirror snapshot to iCloud target when enabled

## State & Backup Strategy (Plain Files)

Default local paths:

- Config root: `~/.pourover/`
- State root: `~/Library/Application Support/PourOver/state/`

Artifacts:

- `lock.json`: canonical resolved state fingerprint
- `last-plan.json`: most recent plan output
- `history/<timestamp>.json`: execution metadata and result summary
- `snapshots/<timestamp>/...`: full snapshot payload

iCloud mirror:

- When enabled, apply triggers snapshot sync to iCloud Drive path.
- Backup command forces snapshot + sync immediately.
- Restore command pulls selected snapshot from local or iCloud mirror.

## Reliability and Error Handling

- Fail-fast and explicit errors for invalid Lua/schema fields
- Atomic write pattern for state files (`*.tmp` then rename)
- Partial-failure reporting with actionable next steps
- Recovery path if brew command fails mid-run:
  - stop remaining dependent operations
  - write failure details in history entry

## Testing Strategy

- Unit tests:
  - manifest validation and normalization
  - uninstall policy resolution
  - planner idempotency and action ordering
- Integration tests:
  - brew adapter (mocked shell interface)
  - file operation execution and rollback-safe behavior
- E2E smoke (macOS):
  - init -> plan -> apply -> plan (no-op) flow

## Observability and UX

- Concise default command output with optional verbose mode
- Exit codes aligned to automation use:
  - `0` success/no-op
  - non-zero for validation, planning, or execution failures
- Machine-readable plan output for future dashboard integration

## Deferred to v2

- Admin dashboard (local web or native macOS)
- Rich drift visualization/history browser
- Multi-profile and host targeting
- Remote sync providers beyond iCloud
