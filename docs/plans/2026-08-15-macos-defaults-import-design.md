# macOS defaults import design

**Status:** Approved  
**Date:** 2026-08-15  
**Command:** `pourover import macos`

## Vision

Systematically read every curated macOS `defaults` key PourOver already knows how to apply, and write declarative Lua so the current Mac’s settings (lock-screen message, trackpad, keyboard, mouse, dock, finder, …) can be reproduced via `pourover apply`.

Example: lock-screen text “Found? Call 513-968-9283” → `macos.defaults.loginwindow.LoginwindowText`.

## Decisions

| Topic | Choice |
|-------|--------|
| Scope | Catalog only (`macos_catalog.yaml`) — expandable by editing the catalog |
| Values | Include every catalog key where `defaults read` returns a value; omit unset (“does not exist”) |
| Merge | Default merge/add-only; `--force` replaces curated `macos.defaults` sections |
| CLI | Dedicated `pourover import macos` (not a flag on packages/files import) |
| Architecture | Catalog → SnapshotDefaults → Format/Merge Lua → `macos.lua` |

## Non-goals

- Dumping non-catalog domains into `custom`
- Interactive key picker / TUI wizard (CLI first; TUI optional follow-up)
- Factory-baseline diffing
- Changing apply semantics for system domains (still need admin)

## UX

```bash
pourover import macos              # merge discovered keys into macos config
pourover import macos --force      # replace curated macos.defaults with snapshot
pourover import macos --dry-run    # preview; write nothing
```

**Artifacts**
- `~/.pourover/macos.lua` — module returning `{ defaults = { … } }` (or top-level shape matching loader expectations)
- Ensure `pourover.lua` requires the module (same pattern as packages)
- Leave `macos.defaults.custom` untouched on merge and force

**Summary output:** counts of keys read, added, skipped (missing/unparseable), note that system-scope keys need admin on apply.

## Architecture

```text
CLI: pourover import macos [--force] [--dry-run]
        │
        ▼
engine.ImportMacOS(opts)
        │
        ├─► config.Catalog() → flatten to DesiredSetting list
        ├─► discovery.SnapshotCatalogDefaults(runner, catalog)
        │      defaults read per key; skip missing; parse typed values
        ├─► configimport.MergeMacOSDefaults / replace on force
        ├─► configimport.FormatMacOSLua → macos.lua
        └─► ensure require("macos") in pourover.lua
```

**Expansion path:** add a section/key to `internal/config/macos_catalog.yaml` → regenerates docs via existing gen tool → automatically importable and appliable.

## Edge cases

| Case | Behavior |
|------|----------|
| Key unset | Omit (not an error) |
| Unparseable value | Skip + warn (verbose); continue |
| System domains | Import; summarize admin note |
| `also_domains` | Try primary domain first; if missing, try alternates |
| `custom` | Never modified |
| No config dir | Error: run `pourover init` (dry-run may still print) |
| Array kinds | Use existing `ParseDefaultsRead` / dock path extraction |

## Testing

- Mocked `DefaultsRunner` snapshot → typed map
- Lua format loads via `LoadManifest` + `Validate`
- Merge vs force unit tests (`custom` preserved)
- CLI dry-run smoke

## Related

- Backlog item: `pourover import --macos` in `docs/v2-backlog.md`
- Catalog: `internal/config/macos_catalog.yaml`
- Apply path: `discovery.DiscoverDefaults` / `exec.ApplyDefaultsWrites`
