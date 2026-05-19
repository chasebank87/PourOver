# PourOver v1 — Micro-Step Execution Guide

**Purpose:** Human-in-the-loop build plan. Each step is small enough to review in one sitting. Do not skip ahead until the **Review gate** for that step is satisfied.

**Companion docs:**
- Design: `2026-05-18-pourover-v1-design.md`
- Original (coarse) plan: `2026-05-18-pourover-v1-implementation.md`

**How to use this doc**
1. Pick the next unchecked step.
2. Implement only that step (or ask the agent: “Do step M2.3 only”).
3. Run the **Verify** command and check **Done when**.
4. Stop at **Review gate** — adjust design or code before continuing.
5. Mark the step done in this file (or your own tracker).

**Agent rule:** When implementing, reference a single step ID (e.g. `M3.2`). Do not implement future steps in the same change.

---

## Design review summary

### What the design gets right
- Clear v1 scope: CLI-only, macOS-only, declarative brew + files.
- Safe-default uninstall with config override (`safe` / `strict` / `non_destructive`).
- Plain-file state + git-friendly workflow + iCloud mirror on apply.
- Clean separation: Lua declares, Go validates/plans/executes.

### Gaps to decide (track in “Open decisions” below)

| Topic | Design says | Decide before step |
|-------|-------------|-------------------|
| Config location | `~/.pourover/` (decided) | **M0.2** — optional `--config` override only |
| Lua embedding | “Go loads Lua” | **M2.1** — `yuin/gopher-lua` vs `layeh/gopher-luar` vs external `lua` binary |
| Lua return shape | Conceptual schema | **M2.3** — table keys and types (document in `docs/config-schema.md`) |
| File operations | links + managed | **M5.1** — symlinks only v1, or copy too? overwrite policy? |
| Brew taps | In apply order | **M6.1** — support taps in v1 or defer? |
| iCloud path | Optional override | **M8.1** — default: `~/Library/Mobile Documents/com~apple~CloudDocs/PourOver/` |
| Exit codes | Mentioned | **M9.4** — document in README |

### Open decisions (fill in as you go)

| ID | Decision | Choice | Date |
|----|----------|--------|------|
| D1 | Config root strategy | Default `~/.pourover/`; state in `~/Library/Application Support/PourOver/state/`; `--config` override. Not `~/.config` (reserved for managed app configs). No dotfiles-repo discovery. | 2026-05-19 |
| D2 | Lua runtime library | `github.com/yuin/gopher-lua` (MIT, pure Go embed) | 2026-05-19 |
| D3 | File ops v1 scope | _TBD_ | |
| D4 | Taps in v1 | _TBD_ | |
| D5 | iCloud default path | _TBD_ | |

---

## Milestone map (high level)

| Milestone | What you can demo | Steps |
|-----------|-------------------|-------|
| **M0** | Empty repo, `go test` passes | M0.1–M0.3 |
| **M1** | `pourover --help` lists commands | M1.1–M1.4 |
| **M2** | Load `pourover.lua` → print JSON manifest | M2.1–M2.6 |
| **M3** | `pourover plan` (brew only, dry text) | M3.1–M3.5 |
| **M4** | `pourover plan` includes files | M4.1–M4.3 |
| **M5** | `pourover apply` installs one formula (real brew) | M5.1–M5.6 |
| **M6** | Safe uninstall prompt works | M6.1–M6.3 |
| **M7** | `pourover init` scaffolds config | M7.1–M7.2 |
| **M8** | State files + history after apply | M8.1–M8.4 |
| **M9** | iCloud mirror after apply | M9.1–M9.3 |
| **M10** | `doctor`, README, CI | M10.1–M10.4 |

---

## M0 — Repository foundation

### M0.1 — Initialize git and module
**Goal:** Version-controlled Go module named for PourOver.

**Do:**
- `git init`
- `go mod init github.com/<you>/pourover` (adjust import path)
- Add `.gitignore` (binaries, `.DS_Store`, state test dirs)

**Verify:** `go mod verify` (or empty module lists cleanly)

**Done when:**
- [ ] `go.mod` exists
- [ ] `git status` is clean after first commit

**Review gate:** Confirm import path matches how you’ll publish/install later.

---

### M0.2 — Config location (D1) ✅ decided
**Goal:** Lock where `pourover.lua` is discovered.

**Decision:**
- **Default config root:** `~/.pourover/` (`pourover.lua` + Lua modules for `require()`)
- **State root:** `~/Library/Application Support/PourOver/state/`
- **Override:** `--config /path/to/pourover.lua` (module path = that file’s directory)
- **Not used:** `~/.config/pourover/` (keeps `~/.config` for managed application configs only)
- **Not used:** walk-up / `~/dotfiles/` repo discovery (users may symlink into `~/.pourover/` if they want)

**Do:** Implement path helpers in **M8.1**; document in `docs/config-schema.md` during **M2.3**.

**Verify:** N/A (decision only)

**Done when:**
- [x] D1 filled in
- [x] Design doc updated

**Review gate:** If you later want `pourover init --path ~/elsewhere`, treat as a new decision — v1 stays `~/.pourover/`.

---

### M0.3 — Project layout stub
**Goal:** Directories exist; no business logic yet.

**Do:** Create empty packages (or `doc.go` only):
- `cmd/pourover/`
- `internal/cli/`
- `internal/config/`
- `internal/discovery/`
- `internal/plan/`
- `internal/exec/`
- `internal/state/`
- `internal/backup/`
- `internal/policy/`
- `test/fixtures/`

**Verify:** `go build ./...` succeeds (may be empty mains)

**Done when:**
- [x] Layout matches design component list

**Review gate:** Rename packages now if you prefer different boundaries.

---

## M1 — CLI skeleton (no Lua, no brew)

### M1.1 — Root command only
**Goal:** Binary runs and prints help.

**Do:**
- `cmd/pourover/main.go` → `cli.Execute()`
- `internal/cli/root.go` with Cobra (or stdlib `flag` if you prefer fewer deps)
- `Use: "pourover"`, `Short` description

**Verify:** `go run ./cmd/pourover --help`

**Done when:**
- [x] Help text mentions macOS declarative env manager

**Review gate:** Cobra vs stdlib — **Cobra** (v1).

---

### M1.2 — Stub subcommands
**Goal:** All v1 commands exist but return “not implemented”.

**Do:** Add stubs: `init`, `plan`, `apply`, `doctor`, `backup`, `restore`

**Verify:** `go run ./cmd/pourover plan` → clear not-implemented message, exit ≠ 0

**Done when:**
- [x] Each subcommand appears in `--help`

**Review gate:** Command names final for v1 (no `sync` alias yet; can add later).

---

### M1.3 — Global flags (minimal)
**Goal:** Shared flags wired but optional for stubs.

**Do:** `--verbose`, `--config` (path), `--json` (for plan output later)

**Verify:** `pourover plan --config /tmp/x` doesn’t panic

**Done when:**
- [x] Flags parse on root or subcommands consistently

**Review gate:** **Persistent flags on root** (`--verbose` / `-v`, `--config`, `--json`); use `cli.Global()` in commands.

---

### M1.4 — CLI test
**Goal:** Prevent regressions on command registration.

**Do:** Test that subcommand names include `plan`, `apply`, `init`.

**Verify:** `go test ./internal/cli/...`

**Done when:**
- [x] Test passes

**Review gate:** Milestone **M1** complete — commit when ready.

---

## M2 — Config: types, Lua load, validation

### M2.1 — Choose Lua embedding (D2)
**Goal:** Pick library; spike loads a table.

**Do:** Tiny spike in `internal/config/lua_spike_test.go` (can delete later):
- Load `return { policy = { uninstall_mode = "safe" } }`
- Read string field in Go

**Verify:** Spike test passes

**Done when:**
- [x] D2 filled in

**Review gate:** **gopher-lua** — MIT license; embed via `lua.NewState()` + `DoString` / future `DoFile`.

---

### M2.2 — Go manifest types
**Goal:** Typed struct for normalized manifest (no loader yet).

**Do:** `internal/config/types.go` — `Manifest`, `Packages`, `Files`, `Policy`, `Backup`

**Verify:** Compiles; optional JSON tags for debug printing

**Done when:**
- [x] Types cover design schema (formulae, casks, links, policy, backup)

**Review gate:** Go uses `snake_case` JSON tags to match Lua keys (`uninstall_mode`, etc.).

---

### M2.3 — Document Lua schema (D3 prep)
**Goal:** Single source of truth for config authors.

**Do:** Create `docs/config-schema.md` with:
- Minimal valid `pourover.lua` example
- Required vs optional keys
- Default for `policy.uninstall_mode` = `"safe"`

**Verify:** You can hand-read example and map to Go types

**Done when:**
- [x] Example file committed under `test/fixtures/config/valid/pourover.lua`

**Review gate:** Approve schema before writing validator (see `docs/config-schema.md`).

---

### M2.4 — Loader: single file, no require
**Goal:** `LoadManifest(path)` returns struct from one `.lua` file.

**Do:** `internal/config/loader.go` — execute file, map Lua table → Go

**Verify:** `go test ./internal/config/ -run LoadManifest_Valid`

**Done when:**
- [x] Valid fixture loads; `policy.uninstall_mode == "safe"`

**Review gate:** Loader returns field paths on type errors (e.g. `packages.formulae[1]`); full validation in M2.5.

---

### M2.5 — Validator
**Goal:** Reject unknown policy modes, empty package names, bad paths.

**Do:** `internal/config/validate.go`

**Verify:** Invalid fixtures fail with path + field in error

**Done when:**
- [x] At least 2 invalid fixture tests

**Review gate:** v1 ignores unknown Lua keys at load; `Validate` enforces semantics on known fields only.

---

### M2.6 — Lua `require()` / modules
**Goal:** Hybrid config: root imports `packages.lua`.

**Do:**
- Adjust loader search path (config directory)
- Fixture: `pourover.lua` + `packages.lua`

**Verify:** Test loads merged formulae from module

**Done when:**
- [x] `require("packages")` works from config dir

**Review gate:** Milestone **M2** complete — commit when ready. Consider `pourover config validate` subcommand later.

---

## M3 — Discovery + plan (brew only)

### M3.1 — Brew runner interface
**Goal:** Testable abstraction over `brew` CLI.

**Do:** `internal/discovery/brew.go` — `Runner` interface, `ExecRunner` impl

**Verify:** Unit test with fake stdout for `brew list --formula`

**Done when:**
- [ ] No real brew calls in unit tests

**Review gate:** Timeout on brew commands?

---

### M3.2 — Parse installed formulae and casks
**Goal:** `DiscoverBrew() → BrewState`

**Do:** Parse `brew list --formula`, `brew list --cask`

**Verify:** Tests with fixture output files

**Done when:**
- [ ] Handles empty lists and multiple lines

**Review gate:** Defer taps (D4) explicitly in doc if skipping.

---

### M3.3 — Plan: brew diff only
**Goal:** Given manifest + brew state → install/remove lists.

**Do:** `internal/plan/brew_diff.go` — actions: `FormulaInstall`, `FormulaRemove`, `CaskInstall`, `CaskRemove`

**Verify:** Table-driven tests: desired ⊃ current → installs only, etc.

**Done when:**
- [ ] Deterministic sort order documented in test

**Review gate:** Case sensitivity for package names?

---

### M3.4 — Plan output formatter
**Goal:** Human-readable plan lines + optional JSON.

**Do:** `internal/plan/render.go`

**Verify:** Snapshot test or golden file for one plan

**Done when:**
- [ ] `plan --json` structure stable enough to document

**Review gate:** JSON field names frozen for future dashboard?

---

### M3.5 — Wire `pourover plan` (brew only)
**Goal:** End-to-end: find config → load → discover → print plan.

**Do:** Implement `plan` command: load `~/.pourover/pourover.lua` by default (or `--config`)

**Verify:** On your Mac with `~/.pourover/pourover.lua`, `pourover plan` prints expected installs

**Done when:**
- [ ] Works with real brew installed
- [ ] Missing `~/.pourover/pourover.lua` → clear error (suggest `pourover init`)

**Review gate:** Commit milestone **M3**. Demo to yourself before apply.

---

## M4 — Files in plan (still no apply)

### M4.1 — Decide file ops scope (D3)
**Goal:** v1 file behavior locked.

**Recommend for simplicity:** symlinks only (`files.links`), no templates in v1.

**Do:** Update `docs/config-schema.md` and D3 row.

**Review gate:** Must decide before M4.2.

---

### M4.2 — Filesystem discovery
**Goal:** Detect existing symlink target vs desired.

**Do:** `internal/discovery/filesystem.go`

**Verify:** Tests with temp dir symlinks

**Done when:**
- [ ] Detects missing link, wrong target, correct link

---

### M4.3 — Plan: merge file actions
**Goal:** Plan order: brew actions then file link actions (or interleave — document choice).

**Do:** Extend `internal/plan` and `plan` command output

**Verify:** `pourover plan` shows file link when fixture config includes links

**Review gate:** Commit milestone **M4**.

---

## M5 — Apply (brew install first, then full brew)

### M5.1 — Dry-run flag
**Goal:** `pourover apply --dry-run` ≡ `plan` (same code path).

**Do:** Shared `buildPlan()` used by both commands

**Verify:** Dry-run performs zero brew subprocesses (mock or count)

**Done when:**
- [ ] Documented in README snippet

---

### M5.2 — Execute formula install only
**Goal:** Apply runs `brew install <formula>` for one missing package.

**Do:** `internal/exec/brew.go` — install path only; no removes yet

**Verify:** Manual test with one new formula in config (pick something small)

**Done when:**
- [ ] Second `plan` shows no install for that formula

**Review gate:** Homebrew PATH / Apple Silicon paths — note in doctor later.

---

### M5.3 — Execute cask install
**Goal:** Same for casks.

**Verify:** Manual or skipped if you don’t want casks in dogfood config

---

### M5.4 — Policy package
**Goal:** `ResolveMode(config, default=safe)` + tests for strict/non_destructive.

**Do:** `internal/policy/mode.go`

**Verify:** `go test ./internal/policy/...`

---

### M5.5 — Execute brew remove with safe prompt
**Goal:** Removes only after single confirmation when mode=safe.

**Do:** `internal/exec/confirm.go` — stdin yes/no; `--yes` flag for CI

**Verify:** Table test for strict (no prompt), safe (prompt), non_destructive (skip removes)

**Review gate:** Wording of prompt — list package names only?

---

### M5.6 — Wire `pourover apply` (brew complete)
**Goal:** Full brew reconcile with policy.

**Verify:** Remove package from lua → plan shows remove → apply prompts → uninstall

**Review gate:** Commit milestone **M5**. **Stop:** real system changes from here on.

---

## M6 — Apply file links

### M6.1 — Execute link / unlink
**Goal:** Create symlinks; remove when no longer desired (respect non_destructive for files?).

**Do:** `internal/exec/files.go`

**Verify:** Temp dir integration test

**Review gate:** Backup before overwriting existing regular file?

---

### M6.2 — Apply ordering
**Goal:** Executor runs brew phase then files (matches design).

**Verify:** Test order of actions in executor

---

### M6.3 — Idempotent apply
**Goal:** Second apply with no config change → no-op message, exit 0.

**Verify:** `go test` + manual double-apply

**Review gate:** Commit milestone **M6**.

---

## M7 — Init scaffolding

### M7.1 — `pourover init` template files
**Goal:** Scaffold `~/.pourover/` with `pourover.lua`, `packages.lua`, and example managed-target entries.

**Do:** Embed templates or `text/template`; create `~/.pourover/` if missing

**Verify:** `pourover init` → `~/.pourover/pourover.lua` exists → `plan` succeeds

---

### M7.2 — Don’t overwrite without flag
**Goal:** `init --force` only overwrites.

**Verify:** Test refuses existing `~/.pourover/pourover.lua` without `--force`

**Review gate:** Commit milestone **M7**.

---

## M8 — Plain-file state

### M8.1 — Paths helper
**Goal:** Resolve default config dir (`~/.pourover/`), state dir (`~/Library/Application Support/PourOver/state/`), and config file from `--config`.

**Do:** `internal/paths/paths.go` (or `internal/config/paths.go`) — overridable in tests

**Verify:** Tests use temp dirs

---

### M8.2 — Atomic JSON writes
**Goal:** `WriteJSONAtomic(path, v)`

**Verify:** Test no partial file on simulated crash (truncate temp)

---

### M8.3 — lock.json + last-plan.json
**Goal:** After successful apply, write manifest hash + plan snapshot.

**Verify:** Files appear after apply; hash changes when config changes

---

### M8.4 — history entries
**Goal:** `history/<iso-timestamp>.json` with success/failure, action summary.

**Verify:** Failed apply still appends history with error

**Review gate:** Commit milestone **M8**.

---

## M9 — iCloud mirror

### M9.1 — Default iCloud path (D5)
**Goal:** Document and implement resolver; respect `backup.icloud.path`.

**Do:** Fill D5; handle “iCloud unavailable” gracefully

**Review gate:** Opt-out via `backup.icloud.enabled = false`

---

### M9.2 — Snapshot directory layout
**Goal:** Copy `state/snapshots/<ts>/` tree to iCloud mirror.

**Verify:** Unit test with two temp roots

---

### M9.3 — Auto-mirror on apply + manual backup/restore
**Goal:** Successful apply triggers mirror; `backup`/`restore` commands work.

**Verify:** Manual: apply → see files in iCloud path; `restore` on test dir

**Review gate:** Commit milestone **M9**. Large snapshot size limits?

---

## M10 — Doctor, docs, CI

### M10.1 — `pourover doctor`
**Goal:** Check brew, config readable, state dir writable, iCloud path reachable if enabled.

**Verify:** `doctor` on clean Mac passes; fails gracefully without brew

---

### M10.2 — README
**Goal:** Install, init, plan, apply, policies, state locations, iCloud.

**Verify:** You can follow README on a fresh clone

---

### M10.3 — Makefile + CI
**Goal:** `make test` runs all packages; GitHub Actions on push.

**Verify:** CI green on main

---

### M10.4 — Exit codes doc (D6)
**Goal:** Document 0/1/2 usage in README.

**Review gate:** Tag v0.1.0-alpha when ready.

---

## Suggested session boundaries (for you + agent)

Use these as natural stopping points when pairing with an agent:

| Session | Steps | You should see |
|---------|-------|----------------|
| 1 | M0 + M1 | `pourover --help` |
| 2 | M2 | Tests load Lua fixture |
| 3 | M3 | `pourover plan` on real Mac |
| 4 | M4 | Plan includes dotfiles |
| 5 | M5 | One real `brew install` via apply |
| 6 | M6 | Symlinks + idempotent apply |
| 7 | M7 | `init` bootstraps `~/.pourover/` |
| 8 | M8–M9 | State files + iCloud copy |
| 9 | M10 | doctor + CI + alpha tag |

---

## What to tell the agent each time

Copy-paste template:

```text
Implement PourOver step [M3.2] only.
- Follow docs/plans/2026-05-18-pourover-v1-micro-steps.md
- Do not implement later steps
- Run the Verify command listed for that step
- Stop and summarize: files changed, test output, anything needing my decision
```

---

## Changes from the coarse implementation plan

The original plan’s **Task 1–10** map to micro-steps roughly as:

| Coarse task | Micro-steps |
|-------------|-------------|
| Task 1 Bootstrap CLI | M0, M1 |
| Task 2 Lua config | M2 |
| Task 3 Discovery | M3.1–M3.2, M4.2 |
| Task 4 Planner | M3.3–M3.5, M4.3 |
| Task 5 Policy | M5.4–M5.5 |
| Task 6 Executor | M5.1–M5.6, M6 |
| Task 7 State | M8 |
| Task 8 iCloud | M9 |
| Task 9 Doctor/e2e | M10.1, M10.2 |
| Task 10 CI | M10.3–M10.4 |

The coarse plan remains useful for TDD *within* a micro-step, but each micro-step should touch **at most 1–3 files** and one behavior.
