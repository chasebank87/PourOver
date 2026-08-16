# MAS install→get fallback Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** On `mas_install`, run `mas install <id>` and if that fails run `mas get <id>` (Homebrew Bundle behavior), wrapping a final failure with App Store sign-in guidance.

**Architecture:** Keep the `mas_install` plan action. Change `InstallMas` in `internal/exec/mas.go` to try install then get via the existing `MasRunner`. Classify “unknown command” for old `mas` so we do not treat a missing `get` as success. Annotate terminal errors with sign-in / first-get guidance. No sudo wrap; upgrades stay `mas upgrade` only.

**Tech Stack:** Go, existing `discovery.MasRunner` fakes in `internal/exec/mas_test.go`.

**Design:** `docs/plans/2026-08-15-mas-get-fallback-design.md`

---

### Task 1: Install succeeds — do not call get

**Files:**
- Modify: `internal/exec/mas_test.go`
- Modify: `internal/exec/mas.go` (later tasks; this task is test-only until Task 2)

**Step 1: Write the failing test**

Extend `recordingMasRunner` so install and get can fail independently. Add:

```go
func TestInstallMas_SkipsGetWhenInstallSucceeds(t *testing.T) {
	runner := &recordingMasRunner{}
	if err := InstallMas(context.Background(), runner, "1518423503"); err != nil {
		t.Fatalf("InstallMas: %v", err)
	}
	if got := strings.Join(runner.calls, ","); got != "install 1518423503" {
		t.Fatalf("calls = %q, want only install", got)
	}
}
```

(`TestInstallMas_RunsMasInstall` already covers this if we keep “install only” as the success path. Prefer renaming that test or adding the explicit skip-get assertion above.)

**Step 2: Run test to verify it fails**

Run: `go test ./internal/exec -count=1 -run TestInstallMas_SkipsGetWhenInstallSucceeds`

Expected: FAIL if the test is new and `InstallMas` already only calls install — this test should **PASS** on current code (documents the fast path). If it passes immediately, that is OK: it locks the “no get on success” contract before we add fallback.

**Step 3: No production change**

**Step 4: Confirm pass**

Run: `go test ./internal/exec -count=1 -run 'TestInstallMas_'`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/exec/mas_test.go
git commit -m "test(exec): lock mas install fast path without get"
```

---

### Task 2: Install failure then get success

**Files:**
- Modify: `internal/exec/mas_test.go`
- Modify: `internal/exec/mas.go`

**Step 1: Write the failing test**

Update `recordingMasRunner.Run` so `failIDs` fails **install** only (not get):

```go
func (r *recordingMasRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	r.calls = append(r.calls, key)
	if len(args) == 2 && args[0] == "install" && r.failIDs[args[1]] {
		return nil, fmt.Errorf("Error: This redownload is not available for this Apple Account")
	}
	if len(args) == 2 && args[0] == "get" && r.failGet[args[1]] {
		return nil, fmt.Errorf("mas get failed")
	}
	if len(args) == 2 && args[0] == "get" && r.noGet {
		return nil, fmt.Errorf("error: Unknown command 'get'")
	}
	// existing uninstall / upgrade failures...
	return nil, nil
}
```

Add `failGet map[string]bool` and `noGet bool` on the struct.

```go
func TestInstallMas_FallsBackToGet(t *testing.T) {
	runner := &recordingMasRunner{failIDs: map[string]bool{"1518423503": true}}
	if err := InstallMas(context.Background(), runner, "1518423503"); err != nil {
		t.Fatalf("InstallMas: %v", err)
	}
	if got := strings.Join(runner.calls, ","); got != "install 1518423503,get 1518423503" {
		t.Fatalf("calls = %q", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/exec -count=1 -run TestInstallMas_FallsBackToGet`

Expected: FAIL (`InstallMas` returns the redownload error; no `get` call)

**Step 3: Write minimal implementation**

In `InstallMas`:

```go
func InstallMas(ctx context.Context, runner discovery.MasRunner, id string) error {
	if _, err := runner.Run(ctx, "install", id); err == nil {
		return nil
	} else if _, getErr := runner.Run(ctx, "get", id); getErr == nil {
		return nil
	} else {
		return fmt.Errorf("mas install %s: %w", id, err) // refine in Task 3–4
	}
	return nil
}
```

Use the **get** error as the returned error when get also fails (Task 3). For this task, returning nil when get succeeds is enough.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/exec -count=1 -run 'TestInstallMas_'`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/exec/mas.go internal/exec/mas_test.go
git commit -m "feat(exec): fall back to mas get after mas install fails"
```

---

### Task 3: Both fail — sign-in guidance

**Files:**
- Modify: `internal/exec/mas_test.go`
- Modify: `internal/exec/mas.go`

**Step 1: Write the failing test**

```go
func TestInstallMas_BothFail_SignInGuidance(t *testing.T) {
	runner := &recordingMasRunner{
		failIDs: map[string]bool{"1518423503": true},
		failGet: map[string]bool{"1518423503": true},
	}
	err := InstallMas(context.Background(), runner, "1518423503")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "App Store") || !strings.Contains(msg, "sign") {
		t.Fatalf("error = %q, want App Store sign-in guidance", msg)
	}
	if !strings.Contains(msg, "1518423503") {
		t.Fatalf("error = %q, want app id", msg)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/exec -count=1 -run TestInstallMas_BothFail_SignInGuidance`

Expected: FAIL (error is a raw mas error without guidance)

**Step 3: Write minimal implementation**

When both fail, wrap:

```go
return fmt.Errorf("mas install %s: %w; sign in to the App Store (Media & Purchases) and retry — first-time apps need mas get after a GUI session (open with: mas open %s)", id, getErr, id)
```

Prefer wrapping `getErr` (last attempt). Include the install error in the chain if cheap (`%w` can only wrap one; join or `%v` the install err in the message).

Suggested:

```go
return fmt.Errorf("mas get %s: %w (after mas install failed: %v); sign in to the App Store (Media & Purchases) and retry. First-time apps must be gotten onto this Apple Account. Open the page with: mas open %s", id, getErr, installErr, id)
```

**Step 4: Run tests**

Run: `go test ./internal/exec -count=1 -run 'TestInstallMas_'`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/exec/mas.go internal/exec/mas_test.go
git commit -m "fix(exec): explain App Store sign-in when mas install and get fail"
```

---

### Task 4: Old mas without get — do not succeed

**Files:**
- Modify: `internal/exec/mas_test.go`
- Modify: `internal/exec/mas.go`

**Step 1: Write the failing test**

```go
func TestInstallMas_GetUnknown_KeepsInstallError(t *testing.T) {
	runner := &recordingMasRunner{
		failIDs: map[string]bool{"1": true},
		noGet:   true,
	}
	err := InstallMas(context.Background(), runner, "1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "redownload") && !strings.Contains(err.Error(), "install") {
		t.Fatalf("error = %q, want original install failure", err)
	}
	if strings.Contains(err.Error(), "Unknown command") && !strings.Contains(err.Error(), "App Store") {
		// still OK if guidance is present; must not return nil
	}
}
```

**Step 2: Run test to verify it fails** (or already passes if Task 3 always returns getErr)

If current code returns get’s “Unknown command” without install context, tighten the assertion: error must mention install failure **or** App Store guidance, and must be non-nil.

**Step 3: Helper**

```go
func masGetUnknown(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unknown command") && strings.Contains(s, "get")
}
```

If `masGetUnknown(getErr)`, return annotated **install** error (not “unknown get”).

**Step 4: Run tests**

Run: `go test ./internal/exec -count=1 -run 'TestInstallMas_|TestApplyMasInstalls_'`

Expected: PASS. Confirm `TestApplyMasInstalls_ContinuesAfterFailure` still works (install fail + get fail for id `1`, get success for id `2`). Update that test’s `calls` to include `get` after each failed install:

Want: `install 1,get 1,install 2` if only `1` is in failIDs (id `2` install succeeds, no get). Current failIDs only `"1"` → calls become `install 1,get 1,install 2`.

**Step 5: Commit**

```bash
git add internal/exec/mas.go internal/exec/mas_test.go
git commit -m "fix(exec): keep mas install error when get is unavailable"
```

---

### Task 5: Docs

**Files:**
- Modify: `README.md` (MAS paragraph around the signed-in sentence)
- Modify: `docs/config-schema.md` (`packages.mas` bullet on install/upgrade)

**Step 1:** Add that apply runs `mas install` then `mas get` on failure; GUI App Store sign-in is required; first-time apps are acquired via `get`.

**Step 2:** No test.

**Step 3:** Commit

```bash
git add README.md docs/config-schema.md
git commit -m "docs: describe mas install fallback to mas get"
```

---

### Task 6: Full test suite

**Step 1:** Run `make test` (unsandboxed; git/sudo tests need host permissions)

Expected: all packages `ok`

**Step 2:** If `TestApplyMasInstalls_*` call counts drifted, fix in this task.

**Step 3:** No extra commit unless fixes were needed.
