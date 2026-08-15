# Trusted Taps + Sudo PAM Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Support nix-darwin-style tap `trusted` flags and `macos.security.pam.sudo_local` (Touch ID / Watch ID / reattach) in PourOver config, plan, and apply.

**Architecture:** Extend packages tap decode to `[]TapSpec` with default `Trusted=true`; gate `brew trust` on the flag. Add `MacOS.Security.PAM.SudoLocal` config → PAM file reconciler (marker-managed `/etc/pam.d/sudo_local`) plus implied formulae and sudo include line.

**Tech Stack:** Go, existing brew plan/exec, Lua decode, admin elevation path used by system defaults

**Design:** `docs/plans/2026-08-15-trusted-taps-and-sudo-pam-design.md`

---

### Task 1: TapSpec types + Lua decode

**Files:**
- Modify: `internal/config/types.go`
- Modify: `internal/config/lua_decode.go` (decode packages taps)
- Modify: `internal/config/validate.go`
- Test: `internal/config/taps_test.go` (create)

**Step 1: Failing tests**

```go
func TestDecodePackages_TapStringTrustedDefault(t *testing.T) { /* taps = { "oven-sh/bun" } → Trusted true */ }
func TestDecodePackages_TapTableTrustedFalse(t *testing.T) { /* { name=..., trusted=false } */ }
func TestDecodePackages_TapTableMissingName(t *testing.T) { /* error */ }
```

Write via temp pourover.lua / LoadManifest or decode helper used by packages tests.

**Step 2:** `go test ./internal/config -run TestDecodePackages_Tap -v` → FAIL

**Step 3: Implement**

```go
type TapSpec struct {
	Name    string
	Trusted bool // default true
}

type Packages struct {
	Taps     []TapSpec `json:"taps,omitempty"`
	Formulae []string  `json:"formulae,omitempty"`
	Casks    []string  `json:"casks,omitempty"`
}

func (p Packages) TapNames() []string { /* map Name */ }
```

Decode: for each taps array element, if string → TapSpec{Name, Trusted:true}; if table → require `name`, optional `trusted` (default true).

Update all call sites that assumed `[]string` taps (`validate`, import format consumers, plan, engine, scaffold comments later). Prefer fixing compile breaks in later tasks where needed; this task should make `go test ./internal/config` pass and `go build ./...` green by updating mechanical `[]string` → `TapNames()` or `TapSpec` loops.

**Step 4:** Tests PASS; `go build ./...`

**Step 5:** Commit `feat(config): TapSpec with optional trusted flag`

---

### Task 2: Gate brew trust on TapSpec.Trusted

**Files:**
- Modify: `internal/plan/brew_diff.go`
- Modify: `internal/exec/brew.go` (`AddTap` signature or options)
- Modify: callers / tests
- Test: `internal/plan/brew_diff_test.go`, `internal/exec/brew_test.go`

**Step 1: Failing tests**

- Plan: desired tap with `Trusted:false`, already tapped, not in TrustedTaps → **no** `tap_trust` action.
- Plan: `Trusted:true`, untrusted → still `tap_trust`.
- `AddTap` with trusted=false → `brew tap` only, no `brew trust`.

**Step 2–4:** Thread `Trusted` through plan desired taps and `AddTap(ctx, runner, name, trusted bool)` (or `AddTapOpts`). Keep official-tap skip.

**Step 5:** Commit `feat(brew): respect tap trusted=false`

---

### Task 3: Import / FormatPackagesLua TapSpec

**Files:**
- Modify: `internal/configimport/packages.go`
- Modify: `internal/configimport/merge.go` if needed
- Modify: engine import + CLI tests that format taps
- Test: `internal/configimport/packages_test.go`

**Step 1:** Format still emits plain quoted strings for taps (implicit trusted). Merge continues on names. Round-trip LoadManifest.

**Step 2–4:** Implement; update any broken format tests.

**Step 5:** Commit `feat(configimport): TapSpec-aware packages.lua format`

---

### Task 4: PAM config types + Lua decode

**Files:**
- Modify: `internal/config/types.go` (`MacOS.Security`)
- Modify: `internal/config/lua_decode.go` (`decodeMacOSTable`)
- Modify: `internal/config/validate.go` if needed
- Test: `internal/config/pam_test.go` (create)

**Step 1: Failing test**

```lua
macos = {
  security = {
    pam = {
      sudo_local = {
        enable = true,
        reattach = true,
        touch_id_auth = true,
        watch_id_auth = true,
      },
    },
  },
}
```

Assert decoded flags. Omitted security → zero value / nil enable unset.

Use pointer or `Enabled *bool` / `Set bool` so omitted vs `enable=false` are distinct:

```go
type SudoLocalPAM struct {
	Configured bool // true if sudo_local table present
	Enable     bool
	Reattach   bool
	TouchIDAuth bool
	WatchIDAuth bool
}
```

**Step 2–4:** Implement decode under `macos.security.pam.sudo_local`.

**Step 5:** Commit `feat(config): macos.security.pam.sudo_local schema`

---

### Task 5: PAM text generation + discovery

**Files:**
- Create: `internal/pam/sudo_local.go` (or `internal/security/pam.go`)
- Test: `internal/pam/sudo_local_test.go`

**Step 1: Failing tests** for `RenderSudoLocal(cfg, reattachPath, watchidPath string) string` including `# pourover: managed` marker and nix-darwin line order.

**Step 2–4:** Implement render; helper to resolve brew prefix paths (`brew --prefix pam-reattach` via runner or `discovery` helper — stubbable). `IsPourOverManaged(content []byte) bool`.

**Step 5:** Commit `feat(pam): render managed sudo_local`

---

### Task 6: Plan + apply PAM reconcile

**Files:**
- Modify: `internal/plan/types.go` (new action types e.g. `pam_sudo_local_write`, `pam_sudo_local_remove`, `pam_sudo_include`)
- Modify: `internal/plan/` brew or new `pam_diff.go`
- Modify: `internal/engine/plan.go`, `apply.go`
- Create: `internal/exec/pam.go`
- Test: plan/exec/engine tests with mem FS or temp dirs (do not touch real `/etc` in unit tests — inject paths)

**Step 1: Failing tests**

- enable+flags → write action + implied formula installs when missing
- enable=false + managed file → remove action
- omitted → no PAM actions
- sudo missing include → include action

**Step 2–4:** Wire after brew formula installs, before or after defaults; admin for `/etc` writes (reuse existing elevation helper from defaults/system scope). Backup pre-existing unmanaged file on first take-over.

**Step 5:** Commit `feat(pam): plan and apply sudo_local`

---

### Task 7: Docs + backlog + README

**Files:**
- Modify: `docs/config-schema.md`
- Modify: `docs/nix-darwin-options.md` (security PAM → mapped; homebrew taps trusted)
- Modify: `docs/v2-backlog.md`
- Modify: `README.md` (brief)

**Step 1:** Document Lua shapes, trust default, PAM flags, admin note.

**Step 2:** `make test && make vet && make build`

**Step 3:** Commit `docs: trusted taps and sudo PAM`

---

## Out of scope (follow-ups)

- Import discovering current PAM into Lua
- TUI settings for PAM/taps
- `system.primaryUser`
