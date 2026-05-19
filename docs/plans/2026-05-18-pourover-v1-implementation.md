# PourOver v1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a macOS-only CLI that declaratively manages Homebrew packages, dotfiles, and config files with safe uninstall defaults and plain-file state plus automatic iCloud snapshot mirroring.

**Architecture:** A Go CLI core loads a Lua-based declarative config, normalizes it into an internal manifest, computes a deterministic reconciliation plan, and executes changes in stable order. State is persisted as plain files and mirrored to iCloud after successful apply runs.

**Tech Stack:** Go (CLI + engine), Lua (config DSL), Homebrew CLI integration, plain JSON state files, macOS filesystem APIs

---

### Task 1: Bootstrap CLI project and command skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/pourover/main.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/commands/init.go`
- Create: `internal/cli/commands/plan.go`
- Create: `internal/cli/commands/apply.go`
- Create: `internal/cli/commands/doctor.go`
- Create: `internal/cli/commands/backup.go`
- Create: `internal/cli/commands/restore.go`
- Test: `internal/cli/root_test.go`

**Step 1: Write the failing test**

```go
func TestRootCommand_HasCoreSubcommands(t *testing.T) {
    cmd := NewRootCommand()
    names := subcommandNames(cmd)
    require.Contains(t, names, "plan")
    require.Contains(t, names, "apply")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestRootCommand_HasCoreSubcommands -v`  
Expected: FAIL with missing command construction

**Step 3: Write minimal implementation**

```go
func NewRootCommand() *cobra.Command {
    cmd := &cobra.Command{Use: "pourover"}
    cmd.AddCommand(newPlanCmd(), newApplyCmd(), newInitCmd(), newDoctorCmd(), newBackupCmd(), newRestoreCmd())
    return cmd
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestRootCommand_HasCoreSubcommands -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add go.mod cmd/pourover/main.go internal/cli internal/cli/commands
git commit -m "feat: scaffold pourover CLI command surface"
```

### Task 2: Implement Lua config loading and manifest validation

**Files:**
- Create: `internal/config/types.go`
- Create: `internal/config/loader.go`
- Create: `internal/config/validate.go`
- Create: `internal/config/fixtures/valid/pourover.lua`
- Create: `internal/config/fixtures/invalid/missing_policy.lua`
- Test: `internal/config/loader_test.go`

**Step 1: Write the failing test**

```go
func TestLoadManifest_ValidFixture(t *testing.T) {
    manifest, err := LoadManifest("internal/config/fixtures/valid/pourover.lua")
    require.NoError(t, err)
    require.Equal(t, "safe", manifest.Policy.UninstallMode)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config -run TestLoadManifest_ValidFixture -v`  
Expected: FAIL with unimplemented loader

**Step 3: Write minimal implementation**

```go
type Manifest struct {
    Packages Packages
    Files    Files
    Policy   Policy
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config -run TestLoadManifest_ValidFixture -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config
git commit -m "feat: add lua manifest loader and schema validation"
```

### Task 3: Build discovery adapters for brew and filesystem state

**Files:**
- Create: `internal/discovery/brew.go`
- Create: `internal/discovery/filesystem.go`
- Create: `internal/discovery/types.go`
- Test: `internal/discovery/brew_test.go`
- Test: `internal/discovery/filesystem_test.go`

**Step 1: Write the failing test**

```go
func TestBrewDiscovery_ParsesInstalledFormulae(t *testing.T) {
    state, err := DiscoverBrewState(fakeRunner("git\nfzf\n", "raycast\n"))
    require.NoError(t, err)
    require.Contains(t, state.Formulae, "git")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/discovery -run TestBrewDiscovery_ParsesInstalledFormulae -v`  
Expected: FAIL with missing parser/discovery

**Step 3: Write minimal implementation**

```go
func DiscoverBrewState(runner Runner) (BrewState, error) { /* parse `brew list` output */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/discovery -run TestBrewDiscovery_ParsesInstalledFormulae -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/discovery
git commit -m "feat: add brew and filesystem state discovery adapters"
```

### Task 4: Implement deterministic planning/diff engine

**Files:**
- Create: `internal/plan/types.go`
- Create: `internal/plan/diff.go`
- Create: `internal/plan/order.go`
- Test: `internal/plan/diff_test.go`

**Step 1: Write the failing test**

```go
func TestBuildPlan_OrdersActionsDeterministically(t *testing.T) {
    plan := BuildPlan(desiredFixture(), currentFixture())
    require.Equal(t, []ActionType{TapAdd, FormulaInstall, CaskInstall, LinkFile}, actionTypes(plan.Actions))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/plan -run TestBuildPlan_OrdersActionsDeterministically -v`  
Expected: FAIL with unimplemented planner/orderer

**Step 3: Write minimal implementation**

```go
func BuildPlan(desired config.Manifest, current discovery.State) Plan { /* diff + stable sort */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/plan -run TestBuildPlan_OrdersActionsDeterministically -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/plan
git commit -m "feat: add deterministic desired-vs-current planner"
```

### Task 5: Add uninstall policy handling and safe confirmation flow

**Files:**
- Create: `internal/policy/mode.go`
- Create: `internal/exec/confirm.go`
- Modify: `internal/cli/commands/apply.go`
- Test: `internal/policy/mode_test.go`
- Test: `internal/exec/confirm_test.go`

**Step 1: Write the failing test**

```go
func TestResolveUninstallPolicy_DefaultsToSafe(t *testing.T) {
    mode := ResolveMode("")
    require.Equal(t, SafeMode, mode)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/policy -run TestResolveUninstallPolicy_DefaultsToSafe -v`  
Expected: FAIL with missing policy resolver

**Step 3: Write minimal implementation**

```go
func ResolveMode(value string) Mode {
    if value == "" { return SafeMode }
    // strict / non_destructive handling
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/policy -run TestResolveUninstallPolicy_DefaultsToSafe -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/policy internal/exec internal/cli/commands/apply.go
git commit -m "feat: implement safe default uninstall policy with overrides"
```

### Task 6: Execute apply actions and ensure idempotent no-op behavior

**Files:**
- Create: `internal/exec/executor.go`
- Create: `internal/exec/actions_brew.go`
- Create: `internal/exec/actions_files.go`
- Test: `internal/exec/executor_test.go`
- Test: `test/e2e/apply_noop_test.go`

**Step 1: Write the failing test**

```go
func TestApply_NoChanges_ReturnsNoOp(t *testing.T) {
    result := Apply(plan.Plan{Actions: nil}, fakeDeps())
    require.True(t, result.NoOp)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/exec -run TestApply_NoChanges_ReturnsNoOp -v`  
Expected: FAIL with missing executor behavior

**Step 3: Write minimal implementation**

```go
if len(p.Actions) == 0 { return Result{NoOp: true}, nil }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/exec -run TestApply_NoChanges_ReturnsNoOp -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/exec test/e2e
git commit -m "feat: add apply executor with deterministic no-op behavior"
```

### Task 7: Persist plain-file state, history, and snapshots

**Files:**
- Create: `internal/state/store.go`
- Create: `internal/state/history.go`
- Create: `internal/state/snapshot.go`
- Test: `internal/state/store_test.go`

**Step 1: Write the failing test**

```go
func TestStateStore_WritesLockAtomically(t *testing.T) {
    err := WriteLock(tempDir, Lock{ManifestHash: "abc"})
    require.NoError(t, err)
    require.FileExists(t, filepath.Join(tempDir, "lock.json"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/state -run TestStateStore_WritesLockAtomically -v`  
Expected: FAIL with missing state writer

**Step 3: Write minimal implementation**

```go
func WriteLock(root string, lock Lock) error { /* write temp + rename */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/state -run TestStateStore_WritesLockAtomically -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/state
git commit -m "feat: add plain-file lock, history, and snapshot storage"
```

### Task 8: Implement iCloud mirror backup and restore flows

**Files:**
- Create: `internal/backup/icloud.go`
- Modify: `internal/cli/commands/apply.go`
- Modify: `internal/cli/commands/backup.go`
- Modify: `internal/cli/commands/restore.go`
- Test: `internal/backup/icloud_test.go`

**Step 1: Write the failing test**

```go
func TestMirrorSnapshotToICloud_CopiesLatestSnapshot(t *testing.T) {
    err := MirrorLatestSnapshot(localRoot, iCloudRoot)
    require.NoError(t, err)
    require.DirExists(t, filepath.Join(iCloudRoot, "snapshots"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/backup -run TestMirrorSnapshotToICloud_CopiesLatestSnapshot -v`  
Expected: FAIL with missing mirror implementation

**Step 3: Write minimal implementation**

```go
func MirrorLatestSnapshot(localRoot, iCloudRoot string) error { /* copy snapshot tree */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/backup -run TestMirrorSnapshotToICloud_CopiesLatestSnapshot -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/backup internal/cli/commands/apply.go internal/cli/commands/backup.go internal/cli/commands/restore.go
git commit -m "feat: add automatic iCloud snapshot mirroring and restore"
```

### Task 9: Add doctor checks and end-to-end smoke validation

**Files:**
- Modify: `internal/cli/commands/doctor.go`
- Create: `internal/doctor/checks.go`
- Create: `test/e2e/smoke_test.go`
- Create: `README.md`

**Step 1: Write the failing test**

```go
func TestDoctor_ReportsMissingBrew(t *testing.T) {
    report := RunChecks(fakeEnvWithoutBrew())
    require.Contains(t, report.Failures, "brew not found")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/doctor -run TestDoctor_ReportsMissingBrew -v`  
Expected: FAIL with missing doctor checks

**Step 3: Write minimal implementation**

```go
func RunChecks(env Env) Report { /* verify brew, lua runtime, writable state paths */ }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/doctor -run TestDoctor_ReportsMissingBrew -v`  
Expected: PASS

**Step 5: Commit**

```bash
git add internal/doctor internal/cli/commands/doctor.go test/e2e README.md
git commit -m "feat: add doctor diagnostics and e2e smoke coverage"
```

### Task 10: Release hardening and first tagged alpha

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `Makefile`
- Modify: `README.md`

**Step 1: Write the failing test/check**

```bash
make test
```

Expected: FAIL until CI/test targets are defined

**Step 2: Run check to verify it fails**

Run: `make test`  
Expected: FAIL with missing target

**Step 3: Write minimal implementation**

```make
test:
	go test ./...
```

**Step 4: Run check to verify it passes**

Run: `make test`  
Expected: PASS

**Step 5: Commit**

```bash
git add .github/workflows/ci.yml Makefile README.md
git commit -m "chore: add ci workflow and alpha release readiness checks"
```
