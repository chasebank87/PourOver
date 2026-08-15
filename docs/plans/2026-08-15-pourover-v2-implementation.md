# PourOver V2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract a reconcile engine, ship a full-control macOS TUI on top of it, then deepen file/dotfile management (managed copies, ownership prune, templates).

**Architecture:** Approach 2 from the V2 design — `internal/engine` owns Plan/Apply/Upgrade/Doctor/Backup/Restore/Import; CLI and Bubble Tea TUI are thin frontends. Files phases extend config/plan/exec/state after the TUI lands. Platform remains macOS-only forever.

**Tech Stack:** Go, Cobra, Bubble Tea / Bubbles / Lip Gloss, existing Lua config (`gopher-lua`), Homebrew CLI, plain JSON state

**Design:** `docs/plans/2026-08-15-pourover-v2-design.md`

**Working branch:** `V2`

---

## Phase 0 — Engine façade

### Task 0.1: Define engine package surface

**Files:**
- Create: `internal/engine/doc.go`
- Create: `internal/engine/options.go`
- Create: `internal/engine/result.go`
- Create: `internal/engine/confirm.go`
- Test: `internal/engine/options_test.go`

**Step 1: Write the failing test**

```go
func TestApplyOptions_Defaults(t *testing.T) {
	opts := ApplyOptions{}
	if opts.AutoYes {
		t.Fatal("AutoYes should default false")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/engine -run TestApplyOptions_Defaults -v`  
Expected: FAIL (package does not exist)

**Step 3: Write minimal types**

```go
// Package engine is the reconcile façade for CLI and TUI frontends.
package engine

type Progress func(line string)

type Confirmer interface {
	Confirm(prompt string) bool
}

type ApplyOptions struct {
	ConfigPath string
	Mode       config.UninstallMode // empty → resolve from manifest
	AutoYes    bool
	Quiet      bool
	DryRun     bool
	Progress   Progress
	Confirm    Confirmer
	Stdout     io.Writer // optional; brew log sink
	Stderr     io.Writer
}

type ApplyResult struct {
	Plan     plan.Plan
	Taps, Formulae, Casks, Removed, Defaults, Linked int
	Renames, Skipped, Failures int
}
```

Wire `config`, `plan` imports; keep types compiling.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/engine -run TestApplyOptions_Defaults -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/engine
git commit -m "feat(engine): scaffold apply options and result types"
```

### Task 0.2: Move plan building behind engine.Plan

**Files:**
- Create: `internal/engine/plan.go`
- Modify: `internal/cli/commands/plan.go` (delegate `buildPlan` / `buildPlanWith`)
- Test: `internal/engine/plan_test.go` (reuse fixtures / mocked runner patterns from `plan_test.go`)

**Step 1: Write failing test** that calls `engine.BuildPlan(ctx, path, runner)` and expects a non-empty or empty plan for a known fixture under `test/fixtures/config`.

**Step 2: Run** `go test ./internal/engine -run TestBuildPlan -v` → FAIL

**Step 3: Implement** `BuildPlan` / `BuildPlanWith` by moving logic from `commands.buildPlanWith` into engine; commands become one-liners.

**Step 4: Run** `go test ./internal/engine ./internal/cli/commands -count=1` → PASS

**Step 5: Commit** `feat(engine): centralize plan building`

### Task 0.3: Move apply execution behind engine.Apply

**Files:**
- Create: `internal/engine/apply.go`
- Modify: `internal/cli/commands/apply_run.go` — `runApply` / `executeApply` / `runApplyActions` call engine
- Test: migrate or wrap key cases from `apply_test.go` to hit engine directly

**Step 1: Failing test** `TestApply_NoChanges` with empty plan / matching state via recording runner.

**Step 2: Run** → FAIL

**Step 3: Move `runApplyActions` body into `engine.Apply`; keep UI session creation in CLI (pass `Progress` + writers). Engine must not import Bubble Tea. CLI still owns fancy `ui.Session` wiring for Phase 0.

**Step 4: Run** `go test ./internal/engine ./internal/cli/commands -count=1` → PASS

**Step 5: Commit** `feat(engine): centralize apply execution`

### Task 0.4: Engine wrappers for upgrade, doctor, backup, restore, import

**Files:**
- Create: `internal/engine/upgrade.go`, `doctor.go`, `backup.go`, `import.go`
- Modify: corresponding `internal/cli/commands/*.go` to delegate
- Test: thin smoke tests per op + existing command tests still pass

**Steps:** For each op — failing smoke test → implement thin wrapper → `go test ./...` → commit per op or one commit `feat(engine): wrap upgrade doctor backup restore import`.

### Task 0.5: Phase 0 docs + parity gate

**Files:**
- Modify: `docs/v2-backlog.md` (note engine extraction started)
- Modify: `README.md` only if CLI UX changed (should not)

**Step 1:** Run full `make test && make vet && make build`  
**Step 2:** Manual smoke: `pourover plan`, `pourover apply --dry-run` on a real config  
**Step 3:** Commit `docs: note V2 phase 0 engine façade`

---

## Phase 1 — TUI shell

### Task 1.1: Add Bubble Tea dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

**Step 1:** `go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest github.com/charmbracelet/lipgloss@latest`  
**Step 2:** `go mod tidy`  
**Step 3:** Commit `chore: add bubbletea stack for V2 TUI`

### Task 1.2: TUI package skeleton + launch rules

**Files:**
- Create: `internal/tui/app.go`, `internal/tui/launch.go`
- Create: `internal/cli/commands/tui.go`
- Modify: `internal/cli/root.go` — no-args interactive → TUI; add `tui` subcommand
- Test: `internal/tui/launch_test.go`, `internal/cli/root_test.go`

**Step 1: Failing tests**

```go
func TestShouldAutoLaunchTUI(t *testing.T) {
	if ShouldAutoLaunch(LaunchEnv{Interactive: false, Args: nil, CI: false}) {
		t.Fatal("non-interactive must not auto-launch")
	}
	if !ShouldAutoLaunch(LaunchEnv{Interactive: true, Args: nil, CI: false}) {
		t.Fatal("interactive no-args should auto-launch")
	}
	if ShouldAutoLaunch(LaunchEnv{Interactive: true, Args: []string{"plan"}, CI: false}) {
		t.Fatal("subcommand must stay CLI")
	}
}
```

**Step 2–5:** Implement → test → wire root → commit `feat(tui): launch rules and tui subcommand`

### Task 1.3: Home model (status + menu)

**Files:**
- Create: `internal/tui/home.go`, `internal/tui/styles.go`
- Test: `internal/tui/home_test.go`

**Behavior:** On init, call `engine.BuildPlan` + lightweight doctor summary; show drift counts; keys navigate menu (Plan, Apply, Upgrade, Doctor, History, Quit).

**Steps:** TDD model Update/View for key navigation → commit `feat(tui): home screen with drift summary`

### Task 1.4: Plan/drift view

**Files:**
- Create: `internal/tui/planview.go`
- Test: `internal/tui/planview_test.go`

**Behavior:** List `plan.Plan` actions grouped; `r` refresh; `a` apply; empty state “in sync.”

**Steps:** TDD → commit `feat(tui): plan and drift view`

### Task 1.5: Apply/Upgrade run view

**Files:**
- Create: `internal/tui/runview.go`
- Create: `internal/tui/confirm.go` (modal Confirmer)
- Modify: engine apply to accept TUI confirmer + progress sink
- Test: `internal/tui/runview_test.go`, `internal/tui/confirm_test.go`

**Behavior:** Reuse phase labels; stream brew lines into viewport; modal for uninstall confirm; cancel stops scheduling new actions.

**Steps:** TDD confirm + progress messages → commit `feat(tui): apply and upgrade run view`

### Task 1.6: Doctor + history list (read-only)

**Files:**
- Create: `internal/tui/doctorview.go`, `internal/tui/historyview.go`
- Test: corresponding `*_test.go`

**Steps:** TDD → commit `feat(tui): doctor and history views`

### Task 1.7: Phase 1 docs + v2.0.0 prep

**Files:**
- Modify: `README.md` (TUI entry, launch rules, mac-only forever)
- Modify: `docs/v2-backlog.md` (mark dashboard/TUI items in progress/done as appropriate)

**Steps:** `make test` → commit `docs: PourOver V2 TUI shell` → tag when ready `v2.0.0`

---

## Phase 2 — TUI complete control

### Task 2.1: Backup / Restore screens

**Files:** `internal/tui/backupview.go`, wire engine backup/restore  
**Test:** model tests for snapshot selection  
**Commit:** `feat(tui): backup and restore`

### Task 2.2: Import screen

**Files:** `internal/tui/importview.go` — flags mirror CLI (`packages`/`files`/`force`/`dry-run`)  
**Commit:** `feat(tui): import`

### Task 2.3: Config iCloud + git screens

**Files:** `internal/tui/configview.go` + engine config helpers if missing  
**Commit:** `feat(tui): config icloud and git`

### Task 2.4: Self-update from TUI

**Files:** menu action → `selfupdate` via engine wrapper  
**Commit:** `feat(tui): self-update action`

### Task 2.5: Opt-in doctor fix actions

**Files:** doctor view “fix” only for safe ops (create state dir, etc.)  
**Commit:** `feat(tui): opt-in doctor fixes`  
**Docs + tag:** `v2.1.0`

---

## Phase 3 — File essentials

### Task 3.1: Schema — `files.managed` + `files.unlink`

**Files:**
- Modify: `internal/config/types.go`, `validate.go`, `lua_decode.go`
- Modify: `docs/config-schema.md`
- Test: `internal/config/validate_test.go`, loader fixtures

**Commit:** `feat(config): files.managed and files.unlink schema`

### Task 3.2: Plan diffs for managed + unlink

**Files:**
- Modify: `internal/plan/types.go`, `file_diff.go`, `render.go`
- Modify: `docs/plan-output.md`
- Test: `internal/plan/file_diff_test.go`

**Commit:** `feat(plan): managed copy and unlink actions`

### Task 3.3: Execute managed copy (atomic) + unlink safeguards

**Files:**
- Modify: `internal/exec/files.go`
- Test: `internal/exec/files_test.go`

**Commit:** `feat(exec): apply managed copies and safe unlinks`

### Task 3.4: Force / backup-on-replace

**Files:**
- Modify: plan + exec file paths; state backup dir under state root `backups/files/`
- Test: replace regular file with backup asserted

**Commit:** `feat(files): force replace with backup`  
**Docs + tag:** `v2.2.0`

---

## Phase 4 — Ownership & prune

### Task 4.1: Persist `owned_files` in lock/snapshot

**Files:** `internal/state/persist.go`, `snapshot.go`, types  
**Test:** round-trip old locks with empty ownership  
**Commit:** `feat(state): track owned file paths`

### Task 4.2: `policy.files_mode` + prune plan actions

**Files:** config policy, `plan/file_diff.go`, policy resolver  
**Commit:** `feat(plan): file prune with files_mode policy`

### Task 4.3: Apply prune via Confirmer

**Files:** `internal/exec/files.go`, engine apply wiring, TUI plan section  
**Commit:** `feat(exec): prune owned undeclared files`  
**Docs + tag:** `v2.3.0`

---

## Phase 5 — Templates

### Task 5.1: Schema `files.templates`

**Files:** config types/validate/docs  
**Commit:** `feat(config): files.templates schema`

### Task 5.2: Sandboxed render + plan unified diff

**Files:**
- Create: `internal/template/render.go` (text/template, allowlisted funcs only)
- Modify: `plan/file_diff.go`, `render.go`
- Test: render + diff tests

**Commit:** `feat(plan): template render diffs`

### Task 5.3: Apply templates as atomic writes

**Files:** `internal/exec/files.go`  
**Commit:** `feat(exec): apply file templates`  
**TUI:** show template section in plan view  
**Docs + backlog triage + tag:** `v2.4.0`

---

## Cross-cutting checklist (every phase)

- [ ] `make vet && make test && make build`
- [ ] No Linux code paths introduced; doctor still fails closed off-mac if ever built
- [ ] CLI exit codes and flags preserved
- [ ] Commit on `V2`; push when asked
- [ ] Update `docs/v2-backlog.md` rows completed this phase

## Out of scope (remain on backlog)

- Web / native dashboard
- Multi-host profiles
- Remote state sync beyond iCloud
- nix-darwin launchd/services/programs
- Strict unknown Lua key errors (unless pulled forward later)
