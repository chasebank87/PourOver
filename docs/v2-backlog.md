# PourOver v2 backlog

Ideas and scope **explicitly deferred from v1**. When we cut v1, anything not done should be listed here so it does not get lost.

**How to use:** Add items with a short rationale and link to the v1 decision or milestone that deferred them. When starting v2 planning, triage this list into milestones.

**Related:** v1 decisions live in `docs/plans/2026-05-18-pourover-v1-micro-steps.md` (Open decisions table). v1 non-goals in `docs/plans/2026-05-18-pourover-v1-design.md`. V2 plan: `docs/plans/2026-08-15-pourover-v2-implementation.md`.

**V2 progress:**
- **Phase 0** (reconcile engine façade) — done. `internal/engine` owns BuildPlan, Apply, UpgradePackages, Doctor, Backup, Restore, and Import; the CLI is a thin frontend.
- **Phase 1** (TUI shell) — done. Interactive no-args / `pourover tui` launch; home drift summary; Plan, Apply, Upgrade, Doctor, and History views.
- **Phase 2** (TUI complete control) — done. Backup/Restore, Import, Config (iCloud + git), self-update from TUI, and opt-in doctor fixes (`f` + y/n; never silent).
- **Phase 3** (file essentials) — done. `files.managed`, `files.unlink`, and `policy.file_replace` (backup-on-replace under `state/backups/files/`).
- **Phase 4** (ownership & prune) — done. `lock.json` `owned_files`, `policy.files_mode`, and apply-time `file_prune` (safe confirm / strict / non_destructive).
- **Phase 5** (templates) — done. `files.templates` sandboxed render, plan unified diffs, atomic apply writes, and ownership tracking.
- **V2 roadmap complete.** Remaining rows below (web dashboard, multi-host profiles, remote state sync beyond iCloud, nix-darwin launchd/services/programs, strict unknown Lua keys, etc.) stay deferred / out of scope for V2.

---

## Files & dotfiles

| Item | Deferred from | Notes |
|------|---------------|-------|
| `files.managed` — copy files to target | D3 (v1 = links only) | **Done (Phase 3)** — atomic copy; `policy.file_replace` for unexpected targets |
| Template / rendered config files | D3 | **Done (Phase 5)** — `files.templates`; sandboxed `text/template`; plan diffs; atomic apply |
| Directory-only links mode | M4 discussion | Restrict links to directories only; stricter validation |
| Auto-unlink / prune undeclared symlinks | M4 discussion | **Done (Phase 4)** — prune PourOver-owned undeclared targets via `owned_files` + `policy.files_mode` |
| `files.unlink` explicit removal list | M4 discussion | **Done (Phase 3)** — safe unlink of files/symlinks; directories refused |
| `--force` when target is a regular file | M4 discussion | **Done (Phase 3)** — `policy.file_replace = "backup"` (`force` alias) |
| Backup-then-replace for blocking targets | M4 discussion | **Done (Phase 3)** — backups under `state/backups/files/<timestamp>/` |
| Allow plan/apply when source missing | M4 discussion | v1: require source to exist at plan time |
| `policy` for file removals (mirror `uninstall_mode`) | M6 review gate | **Done (Phase 4)** — `files_mode`: safe / strict / non_destructive for prune |

## Homebrew

| Item | Deferred from | Notes |
|------|---------------|-------|
| Homebrew **taps** in discovery/plan/apply | D4 | **Done** — `packages.taps`; import/plan/apply; core taps never untapped |
| Brew bundle / Brewfile import | — | Optional input format |
| `brew pin` / version pinning | — | Nix-like pinning is out of scope for v1 |

## Config & UX

| Item | Deferred from | Notes |
|------|---------------|-------|
| `pourover config validate` subcommand | M2 review gate | Validate without discover/apply |
| Strict mode: error on unknown Lua keys | M2.5 review gate | v1 ignores unknown keys |
| Dotfiles repo discovery (`~/dotfiles`, walk-up to git root) | D1 | v1 uses `~/.pourover/` only; users can symlink |
| `pourover init --path` custom config root | M0.2 | v1 fixed to `~/.pourover/` |
| `sync` alias for `apply` | M1.2 | Naming only |
| Multi-profile / host-specific manifests | Design v2 | e.g. `policy.hosts["work-laptop"]` |
| `pourover init` from remote template URL | — | Bootstrap from community presets; partial overlap with `config git restore` |

## State, backup & platform

| Item | Deferred from | Notes |
|------|---------------|-------|
| Default iCloud path formalized + override UX | D5 | v1 will pick a default at M9; document here when chosen |
| Remote sync beyond iCloud (S3, git remote **state**) | Design v2 | Git **config** sync (`pourover config git`) landed; state remotes still deferred |
| Rich history browser / drift UI | Design v2 | **Phase 1 done** — TUI History + Plan/drift views; deeper browse still optional |
| Rollback command beyond `restore` | — | One-click undo last apply |

## Dashboard & tooling

| Item | Deferred from | Notes |
|------|---------------|-------|
| Interactive TUI (Bubble Tea) | Design v2 | **Phase 2 done** — full control surface including Backup/Import/Config/self-update and opt-in doctor fixes |
| Admin dashboard (local web or native macOS) | Design v1 non-goals | Plan JSON shape frozen; TUI is the primary UI for V2 |
| `doctor` fix mode (auto-repair) | — | **Done (opt-in)** — TUI `f` + y/n for safe ops (state dir, config scaffold); tips only for brew/iCloud/git |
| CI / headless apply with `--yes` | M5 | Non-interactive installs/uninstalls; TUI never auto-launches when `CI=true` or non-TTY |

## Platform

| Item | Deferred from | Notes |
|------|---------------|-------|
| Linux / non-macOS support | Design v1 non-goals | **macOS-only forever** — not a deferred port |
| Full Nix-like store / build graph | Design v1 non-goals | Declarative reconcile only |

## macOS defaults (follow-ups)

| Item | Deferred from | Notes |
|------|---------------|-------|
| Wallpaper (desktoppr / osascript / Wallpaper framework) | nix-darwin parity cut | nix-darwin has no first-class wallpaper either |
| Finder sidebar Favorites management | nix-darwin#1663 | Opaque sidebar plists; not simple `defaults` |
| ByHost control-center (full UUID path) | nix-darwin controlcenter | Catalog writes `com.apple.controlcenter`; ByHost path not used |
| CustomSystemPreferences (sudo `/Library/Preferences` escape hatch) | nix-darwin | `loginwindow`/`smb`/`SoftwareUpdate` are named sections; arbitrary system domains still user-only via `custom` |
| `pourover import macos` snapshot curated keys | — | **Done** — curated catalog → `macos.lua`; `--force` / `--dry-run`; expand via `macos_catalog.yaml` |
| nix-darwin `launchd` / `services` / `programs` / PAM | MyNixOS full tree | Indexed in `docs/nix-darwin-options.md` |

---

## v1 locked choices (for contrast)

See **D3** in micro-steps. v1 file behavior:

- **`files.links` only** (symlinks)
- Source paths relative to the directory containing `pourover.lua`
- Targets support `~` expansion; compare canonical absolute paths
- No automatic unlink of undeclared symlinks
- Existing non-symlink at target → **error** in plan (no force)
- Source must exist at plan time
- Plan/apply order: brew actions, then macOS defaults, then file link actions

---

## Adding items

When deferring work from a PR or micro-step, add a row above with:

1. **What** — one line
2. **Deferred from** — decision ID, milestone, or PR link
3. **Notes** — constraints or dependencies
