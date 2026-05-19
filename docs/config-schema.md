# PourOver config schema (v1)

Config lives in `~/.pourover/` by default (`pourover.lua` + optional Lua modules). Go loads the Lua table and normalizes it into a `Manifest` (see `internal/config/types.go`).

## Top-level keys

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `packages` | no | table | Homebrew formulae and casks |
| `files` | no | table | Dotfiles and other paths to reconcile |
| `policy` | no | table | Safety and behavior options |
| `backup` | no | table | Snapshot / iCloud settings |

Empty or omitted sections are treated as empty lists / defaults.

Unknown top-level or nested keys in Lua are ignored for v1 (not an error). Semantic validation runs after load via `Validate`.

## `packages`

| Key | Type | Description |
|-----|------|-------------|
| `formulae` | array of strings | Homebrew formulae to install |
| `casks` | array of strings | Homebrew casks to install |

## `files`

| Key | Type | Description |
|-----|------|-------------|
| `links` | array of tables | Symlinks: `source` → `target` |
| `managed` | array of tables | Files to copy/template (v1 may implement links only first) |

Each link / managed entry:

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `source` | yes | string | Path relative to config dir or absolute |
| `target` | yes | string | Path on disk (may use `~` for home) |

## `policy`

| Key | Required | Type | Default | Values |
|-----|----------|------|---------|--------|
| `uninstall_mode` | no | string | `"safe"` | `safe`, `strict`, `non_destructive` |

- **safe** — prompt before uninstalling undeclared Homebrew packages
- **strict** — uninstall undeclared packages without prompting
- **non_destructive** — never uninstall undeclared packages

## `backup`

| Key | Type | Description |
|-----|------|-------------|
| `icloud` | table | iCloud Drive mirror settings |

### `backup.icloud`

| Key | Required | Type | Default | Description |
|-----|----------|------|---------|-------------|
| `enabled` | no | boolean | `false` | Mirror snapshots to iCloud after successful apply |
| `path` | no | string | (built-in default) | Override iCloud destination directory |

## Minimal example

See `test/fixtures/config/valid/pourover.lua`.

## Module layout (hybrid config)

Root `pourover.lua` may import modules from the same directory:

```lua
local packages = require("packages")

return {
  packages = packages,
  policy = { uninstall_mode = "safe" },
}
```

`require("name")` resolves to `name.lua` next to the root config file.
