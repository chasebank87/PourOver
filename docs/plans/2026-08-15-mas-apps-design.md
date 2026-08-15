# Mac App Store (`mas`) apps design

**Status:** Approved  
**Date:** 2026-08-15

## Vision

Declarative Mac App Store installs by numeric ID (nix-darwin `homebrew.masApps` parity), with PourOver-style reconcile: install, uninstall under `uninstall_mode`, upgrade, and import.

## Decisions

| Topic | Choice |
|-------|--------|
| Mechanism | Call `mas` CLI directly (not Brewfile / `brew bundle`) |
| Config shape | nix-darwin map: display name → positive integer ID |
| Identity | Match and act by **ID only**; name is display/plan label |
| Uninstall | Yes — same `policy.uninstall_mode` as formulae/casks |
| Implied formula | Non-empty managed `mas` → ensure `mas` formula in brew plan |
| Upgrade | `pourover upgrade` includes declared outdated MAS apps |
| Import | `--packages` merges `mas list` into `packages.mas` |

## Non-goals

- Purchasing unpaid apps via `mas purchase` / `mas get` automation beyond install of already-gotten apps
- Treating MAS apps as Homebrew casks (no cross-type declare)
- Running `mas` under sudo (user session only — avoids nix-darwin `installd` session bugs)

## Config shape

```lua
packages = {
  formulae = { "git" },
  casks = { "raycast" },
  mas = {
    Xcode = 497799835,
    ["1Password for Safari"] = 1569813296,
  },
}
```

**Presence:**

- **`mas` key omitted** → unmanaged (no MAS discovery, install, remove, upgrade, or import writes).
- **`mas` key present** (including `mas = {}`) → managed; desired set is the map entries; empty means desire zero App Store apps.

**Validation:**

- Values must be positive integers.
- Duplicate IDs → load error.
- Duplicate names → load error.
- Keys are non-empty strings (display names).

Decoded as `MasConfigured bool` + `[]MasApp{Name, ID}`, sorted by ID for stable plans.

## Architecture

```text
packages.mas (Lua map)
  → ExpandMasFormulae (append "mas" formula when managed && len>0 or always when managed? → when managed and (len>0 OR we need mas for list/outdated during plan))
  → DiscoverMas: `mas list` / `mas outdated`
  → BuildMasPlan / BuildUpgradePlan mas_upgrade
  → apply: mas install|uninstall|upgrade <id>
  → import: MergeMasApps into packages.lua
```

**Implied `mas` formula:** When `MasConfigured` and (desired apps non-empty **or** plan needs discovery that requires the binary), ensure formula `mas` is in the brew desired set before `BuildBrewPlan` (same pattern as `ExpandPAMFormulae`). Practically: whenever `MasConfigured`, expand so discovery/apply can run.

**Discovery:** `mas list` → lines `ID Name…`. `mas outdated` → pending update IDs (declared ∩ outdated).

**Plan actions:**

| Action | Apply |
|--------|--------|
| `mas_install` | `mas install <id>` |
| `mas_remove` | `mas uninstall <id>` |
| `mas_upgrade` | `mas upgrade <id>` |

`Action.Name` = display name; `Action.Value` = decimal ID string.

**Order (apply):** formula installs (incl. implied `mas`) → cask installs → **mas installs** → removes (brew then mas, gated by uninstall_mode) → files/defaults/PAM as today. Upgrades stay on the upgrade command path.

**Uninstall policy:** Undeclared IDs from `mas list` when managed:

- `safe` — plan `mas_remove`, prompt once
- `strict` — remove without prompt
- `non_destructive` — never plan MAS removes

**Upgrade:** Only IDs in desired `packages.mas` that appear in `mas outdated`.

**Import:** With packages import enabled and `mas` available:

- Merge by ID; add-only without `--force`; `--force` replaces MAS map with discovered set (aligned with other package force semantics where applicable).
- Lua key = name from `mas list`; value = ID.
- If ID already declared under another name, keep existing name.

## Edge cases

| Case | Behavior |
|------|----------|
| Config name ≠ App Store name, same ID | Installed; plan uses config name |
| Not signed into App Store | Install/upgrade fails with sign-in guidance |
| `mas` binary missing | Formula install first; if still missing, clear error |
| App gotten but not installed | `mas install` |
| Last app removed → `mas = {}` | Managed empty → uninstall undeclared (all listed) under policy |
| MAS + cask same product | Independent; no special-case |

## Testing

- Lua decode/validate: map, bad IDs, duplicates, omitted vs empty configured
- Expand implies `mas` formula when configured
- Diff: install missing IDs; remove undeclared under safe/strict; skip under non_destructive
- Upgrade: declared ∩ outdated only
- Import merge add-only / force
- Fake `mas` runner + stdout fixtures; no live App Store in CI

## Reference

- nix-darwin `homebrew.masApps` → Brewfile `mas "Name", id: N` via `brew bundle`
- [mas-cli/mas](https://github.com/mas-cli/mas)
