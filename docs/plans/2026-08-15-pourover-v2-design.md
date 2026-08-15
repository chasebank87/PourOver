# PourOver V2 Design

**Status:** Approved  
**Date:** 2026-08-15  
**Branch:** `V2`  
**Platform:** macOS only (permanent — not deferred; never Linux)

## Vision

PourOver remains a declarative macOS environment manager (Homebrew, defaults, files). V2 makes humans drive the system from a **full-control terminal UI**, while scripts/CI keep using the **CLI**. Both frontends call one **reconcile engine**. After engine + TUI ship, V2 deepens **file/dotfile** management into a full toolkit (managed copies, ownership prune, templates, file policy).

## Product decisions

| Topic | Choice |
|-------|--------|
| North star | Visibility (TUI) first, then dotfile maturity |
| Visibility form | Terminal TUI (not web, not native app) |
| TUI depth | Full control surface (plan, apply, upgrade, doctor, restore, backup, import, config, self-update) |
| Files depth | Full toolkit: managed + force/backup + unlink + owned prune + templates + `policy.files_mode` |
| Architecture | Extract reconcile engine, then dual thin frontends (Approach 2) |
| Platform | macOS only forever |

## Non-goals

- Linux / non-macOS
- Full Nix-like store / build graph
- Remote multi-host orchestration
- Native Swift/AppKit app
- Local web dashboard (plan JSON remains available; no web UI in V2)
- Silent auto-repair in doctor (opt-in fix actions only)

## Phased roadmap

| Phase | Name | Outcome | Suggested release |
|-------|------|---------|-------------------|
| **0** | Engine façade | Stable in-process API; CLI becomes thin caller; no user-visible behavior change | `v2.0.0-engine` / pre-release |
| **1** | TUI shell | Interactive home: status, plan/drift, apply/upgrade with live progress, doctor, history | **v2.0.0** (first user-facing V2) |
| **2** | TUI complete control | Restore, backup, import, config git/iCloud, self-update from TUI; same safety as CLI | `v2.1.0` |
| **3** | File essentials | `files.managed`, force/backup-on-replace, explicit `files.unlink` | `v2.2.0` |
| **4** | Ownership & prune | Owned-path tracking; safe auto-unlink; `policy.files_mode` | `v2.3.0` |
| **5** | Templates | `files.templates` with sandboxed render + plan diffs | `v2.4.0` |

**V2 done when:** daily Mac setup is TUI-driven; file links/copies/templates are safe and pruneable; CLI remains first-class for headless use; docs/CI state mac-only permanently.

## Architecture

```text
┌─────────────────────────────────────────────┐
│  Frontends                                  │
│  ┌──────────────┐      ┌─────────────────┐  │
│  │ CLI (cobra)  │      │ TUI (Bubble Tea)│  │
│  │ scripts/CI   │      │ interactive Mac │  │
│  └──────┬───────┘      └────────┬────────┘  │
│         │                       │           │
│         └───────────┬───────────┘           │
│                     ▼                       │
│         ┌───────────────────────┐           │
│         │  internal/engine      │  Phase 0  │
│         │  Plan / Apply / …     │           │
│         │  Events + Progress    │           │
│         └───────────┬───────────┘           │
└─────────────────────┼───────────────────────┘
                      ▼
        config → discovery → plan → exec → state/backup
```

### Engine façade (`internal/engine`)

- Ops API: load manifest, build plan, apply, upgrade, doctor, backup, restore, import, config helpers.
- Returns structured results (plans, summaries, diagnostics) — not pre-formatted strings.
- Progress via callbacks so CLI and TUI render differently.
- Policy prompts via `Confirmer` interface (CLI stdin vs TUI modal).
- Phase 0 is behavior-preserving refactor; existing tests migrate to call the engine.

### CLI

- Automation surface: `--json`, `--yes`, `--quiet`, `--dry-run`, exit codes unchanged.
- Thin: parse flags → engine → print.
- No TUI dependency on non-interactive paths.

### TUI

- Stack: Bubble Tea + Bubbles + Lip Gloss (aligns with Go and existing `internal/ui` styling).
- Launch rules:
  - Interactive TTY + no args → TUI home
  - `pourover tui` → always TUI
  - Any subcommand → CLI
  - Non-TTY / `CI=true` / piped → never auto-TUI
- Never owns brew/file logic — only calls engine and renders.
- Live apply/upgrade parks progress chrome before brew logs / Password prompts (same as V1 session UI).

## TUI UX (Phases 1–2)

**Home:** config path, drift counts, last apply, doctor health; actions for Plan, Apply, Upgrade, Doctor, History, Backup/Restore, Import, Config, Quit.

**Plan/drift:** grouped actions (taps → formulae → casks → defaults → files → advisories). Refresh, filter, apply, copy JSON. Empty = in sync.

**Run view:** phase progress + scrollable brew logs; safe uninstall → modal `Confirmer`; soft-fail semantics preserved.

**History:** browse history + snapshots; open detail; jump to Restore.

**Doctor:** pass/warn/fail checklist; Phase 2 may offer opt-in fix actions (never silent).

**Config:** iCloud + git sync via engine wrappers of existing config commands.

**Safety:** TUI never bypasses policy. Session-scoped “auto-confirm uninstalls” toggle defaults off (V2 does not persist this unless added later).

## File & dotfile model (Phases 3–5)

### Keep from V1

- `files.links` only until Phase 3 extends the schema
- Sources relative to config root; `~` in targets
- Order: brew → defaults → files
- No silent deletes outside policy

### Phase 3 — Essentials

| Capability | Behavior |
|------------|----------|
| `files.managed` | Copy source → target; plan diffs by content/hash; atomic write |
| Force / backup-on-replace | Existing regular file / wrong link → error unless force; backup to state `backups/files/…` then replace |
| `files.unlink` | Explicit paths removed only if PourOver-owned or safe symlinks into the config tree |

### Phase 4 — Ownership & prune

- Record owned paths in lock/snapshot after successful link/managed apply
- Undeclared **owned** paths are prune candidates
- `policy.files_mode`: `safe` \| `strict` \| `non_destructive` (mirror package uninstall modes)
- Never prune paths PourOver did not create

### Phase 5 — Templates

- `files.templates`: source + target; render with fixed context (`hostname`, `user`, `home`, allowlisted env)
- Sandboxed text templates only (no arbitrary code execution)
- Plan shows rendered unified diff; apply writes like `managed`

## Errors, testing, migration

**Errors**

- Engine returns typed errors; frontends map to exit codes / TUI banners
- Partial apply continues later phases; history records failures
- File ops never delete non-owned paths; backup-before-replace required when force replaces a regular file
- TUI cancel: stop scheduling new actions; in-flight brew may finish; history marks interrupted

**Testing**

- Phase 0: engine parity with existing CLI behavior
- TUI: Bubble Tea model unit tests (keys, confirm); no full terminal e2e required in CI
- Files: managed/hash, ownership prune, template render + plan diff; mocked FS
- CI remains darwin; doctor/docs assert mac-only

**Migration**

- Existing `~/.pourover` Lua keeps working; new keys are additive
- Unknown-key strict mode stays backlog (not required for V2 done)
- `lock.json` gains `owned_files`; old locks load with empty ownership (conservative prune)
- Semver: user-facing TUI ships as **v2.0.0**; file phases follow as minor bumps

## Docs to update per phase

- README (TUI entry, mac-only forever)
- `docs/config-schema.md`, `docs/plan-output.md`
- Triage `docs/v2-backlog.md` after each phase (mark done / leave deferred)

## Related

- V1 design: `docs/plans/2026-05-18-pourover-v1-design.md`
- V2 backlog (source of deferred items): `docs/v2-backlog.md`
- Implementation plan: `docs/plans/2026-08-15-pourover-v2-implementation.md`
