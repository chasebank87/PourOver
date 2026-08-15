# PourOver config schema (v1)

Config lives in `~/.pourover/` by default (`pourover.lua` + optional Lua modules). Go loads the Lua table and normalizes it into a `Manifest` (see `internal/config/types.go`).

## Top-level keys

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `packages` | no | table | Homebrew taps, formulae, and casks |
| `files` | no | table | Dotfiles and other paths to reconcile |
| `policy` | no | table | Safety and behavior options |
| `backup` | no | table | Snapshot / iCloud settings |
| `macos` | no | table | macOS `defaults` preferences (nix-darwin-style) |

Empty or omitted sections are treated as empty lists / defaults.

Unknown top-level or nested keys in Lua are ignored for v1 (not an error). Semantic validation runs after load via `Validate`.

## `packages`

| Key | Type | Description |
|-----|------|-------------|
| `taps` | array of strings | Homebrew taps to ensure are present (`owner/repo`) |
| `formulae` | array of strings | Homebrew formulae to install |
| `casks` | array of strings | Homebrew casks to install |

Names must be **lowercase Homebrew tokens** (`"raycast"`, not `"Raycast"`; `"homebrew/cask-fonts"`, not `"Homebrew/Cask-Fonts"`). Capital letters fail `Validate` / `pourover doctor` / `plan` / `apply` so a mistyped token cannot install under one casing and then look undeclared later.

Apply order: **tap adds (with `brew trust --tap` for non-official taps, then `brew update`) → trust already-tapped untrusted taps → formula/cask installs → removes** (untap follows `policy.uninstall_mode`; `homebrew/core` and `homebrew/cask` are never untapped). Official `homebrew/*` taps are always trusted by Homebrew and are not passed to `brew trust`. After any new tap, PourOver runs `brew update` once so packages from that tap are installable.

`pourover import --packages` merges `brew tap` / `brew list` into `packages.lua` (add-only by default; `--force` replaces). Core taps (`homebrew/core`, `homebrew/cask`) are omitted from import output.

## `files`

| Key | Type | v1 | Description |
|-----|------|-----|-------------|
| `links` | array of tables | **supported** | Symlinks: `source` → `target` |
| `managed` | array of tables | **reserved** | Parsed but not planned/applied in v1 (see `docs/v2-backlog.md`) |

### `files.links` (v1)

Each entry:

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `source` | yes | string | Path relative to the directory containing `pourover.lua`, or absolute |
| `target` | yes | string | Path on disk (`~` expanded) |

**v1 behavior:**

- Plan/apply only creates or updates symlinks for declared links.
- `source` must exist when planning.
- If `target` exists and is not a symlink → error (no overwrite).
- No automatic removal of symlinks absent from config.
- Typical layout: sources under `~/.pourover/config/...`, targets under `~/.config/...`.

`pourover import --files` copies existing home/`~/.config` paths into the config tree, writes `files.links`, and retargets live paths to symlinks (use `--force` if links are already declared).

Example:

```lua
files = {
  links = {
    { source = "config/nvim", target = "~/.config/nvim" },
  },
}
```

## `policy`

| Key | Required | Type | Default | Values |
|-----|----------|------|---------|--------|
| `uninstall_mode` | no | string | `"safe"` | `safe`, `strict`, `non_destructive` |

- **safe** — prompt before uninstalling undeclared Homebrew packages/taps (only formulae installed on request; dependency-only packages are ignored; `homebrew/core` / `homebrew/cask` never untapped). A package listed under `formulae` or `casks` counts as declared for either type.
- **strict** — uninstall undeclared packages/taps without prompting
- **non_destructive** — never uninstall undeclared packages/taps

## `backup`

| Key | Type | Description |
|-----|------|-------------|
| `icloud` | table | iCloud Drive mirror settings for **state** snapshots |
| `git` | table | Git remote sync for **config** (`~/.pourover` as the repo) |

### `backup.icloud`

| Key | Required | Type | Default | Description |
|-----|----------|------|---------|-------------|
| `enabled` | no | boolean | `false` | Mirror snapshots to iCloud after successful apply |
| `path` | no | string | `~/Library/Mobile Documents/com~apple~CloudDocs/PourOver/` | Override iCloud destination directory |

Enable via CLI: `pourover config icloud enable` (optional `--path`).

### `backup.git`

| Key | Required | Type | Default | Description |
|-----|----------|------|---------|-------------|
| `enabled` | no | boolean | `false` | Treat config dir as a git repo and sync to `remote` |
| `remote` | no | string | (empty) | Git remote URL (e.g. `git@github.com:USER/pourover-config.git`) |
| `auto_push` | no | boolean | `true` when `git` table is present | After successful apply/import, commit+push when dirty (soft-fail) |
| `branch` | no | string | `"main"` | Branch to push/pull |

Setup: `pourover config git setup <url>`. Emergency restore: `pourover config git restore <url>`.

## `macos`

Declarative preferences via `defaults write` (nix-darwin `system.defaults`). **Unset keys are unmanaged.**

Searchable key list and Lua syntax: [`docs/macos-defaults.md`](macos-defaults.md).
Full nix-darwin option tree: [`docs/nix-darwin-options.md`](nix-darwin-options.md).

### `macos.defaults`

Named tables match nix-darwin sections (`dock`, `finder`, `NSGlobalDomain`, `screencapture`, `trackpad`, …). `custom` is `CustomUserPreferences` (any domain).

Scalar types: **boolean**, **integer**, **float**, **string**. Hyphenated keys: `["show-recents"] = false`.

Unknown section or key → config error (see the catalog). Arbitrary domains go under `custom`.

Example:

```lua
macos = {
  defaults = {
    dock = { autohide = true, ["show-recents"] = false },
    finder = { ShowPathbar = true },
    screencapture = { type = "png" },
    custom = {
      ["com.apple.Safari"] = { ShowFullURLInSmartSearchField = true },
    },
  },
}
```

`loginwindow` / `smb` / `SoftwareUpdate` write machine plists (admin). Wallpaper, Finder sidebar Favorites, and Dock `persistent-apps` are not applied.

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
