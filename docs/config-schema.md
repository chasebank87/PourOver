# PourOver config schema

Config lives in `~/.pourover/` by default (`pourover.lua` + optional Lua modules). Go loads the Lua table and normalizes it into a `Manifest` (see `internal/config/types.go`).

## Top-level keys

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `packages` | no | table | Homebrew taps, formulae, casks, and Mac App Store apps |
| `files` | no | table | Dotfiles and other paths to reconcile |
| `policy` | no | table | Safety and behavior options |
| `backup` | no | table | Snapshot / iCloud settings |
| `macos` | no | table | macOS `defaults` preferences and optional `security.pam.sudo_local` |

Empty or omitted sections are treated as empty lists / defaults.

Unknown top-level or nested keys in Lua are ignored (not an error). Semantic validation runs after load via `Validate`.

## `packages`

| Key | Type | Description |
|-----|------|-------------|
| `taps` | array of strings or tables | Homebrew taps to ensure are present (`owner/repo`); see below |
| `formulae` | array of strings | Homebrew formulae to install |
| `casks` | array of strings | Homebrew casks to install |
| `mas` | table of name → integer | Mac App Store apps (`packages.mas`); omit to leave App Store unmanaged |

### `packages.taps`

Each entry is either:

- a **string** tap name (`"owner/repo"`) — treated as `{ name = "…", trusted = true }`
- a **table** `{ name = "owner/repo", trusted = true|false }` — `trusted` defaults to **`true`** when omitted

`trusted = false` still ensures the tap is present (`brew tap`) but skips `brew trust --tap` and does not emit `tap_trust` drift for that tap.

Names must be **lowercase Homebrew tokens** (`"raycast"`, not `"Raycast"`; `"homebrew/cask-fonts"`, not `"Homebrew/Cask-Fonts"`). Capital letters fail `Validate` / `pourover doctor` / `plan` / `apply` so a mistyped token cannot install under one casing and then look undeclared later.

Apply order: **tap adds (with `brew trust --tap` only when `trusted` and the tap needs explicit trust, then `brew update`) → trust already-tapped untrusted taps (same gate) → formula/cask installs → removes** (untap follows `policy.uninstall_mode`; `homebrew/core` and `homebrew/cask` are never untapped). Official `homebrew/*` taps are always trusted by Homebrew and are not passed to `brew trust`. After any new tap, PourOver runs `brew update` once so packages from that tap are installable.

```lua
packages = {
  taps = {
    "homebrew/cask-fonts",
    { name = "oven-sh/bun", trusted = true },
    { name = "heroku/brew", trusted = false },
  },
  formulae = { "git" },
  casks = { "raycast" },
  mas = {
    Xcode = 497799835,
  },
}
```

`pourover import --packages` merges `brew tap` / `brew list` into `packages.lua` (add-only by default; `--force` replaces). Import emits plain tap **strings** (implicit `trusted = true`). Core taps (`homebrew/core`, `homebrew/cask`) are omitted from import output. When `mas` is available, import also writes `packages.mas` from `mas list`.

### `packages.mas`

Name → numeric App Store ID map (nix-darwin `homebrew.masApps` style).

- **Omit** `mas` — App Store apps are unmanaged (no install/remove/upgrade actions).
- **`mas = {}`** — manage MAS and desire **zero** apps (undeclared installed apps follow `policy.uninstall_mode`).
- Declaring `mas` (even empty) implies the Homebrew formula `mas`.
- Install/upgrade needs you signed into the App Store.

IDs must be positive integers. Duplicate names or IDs fail validation. Declaring `mas` does not treat those apps as casks.

## `files`

| Key | Type | Status | Description |
|-----|------|--------|-------------|
| `links` | array of tables | **supported** | Declared files: `source` → `target` (activated as regular file copies) |
| `managed` | array of tables | **supported** | Copy source → target (`source`, `target`) |
| `templates` | array of tables | **supported** | Render text/template source → target (`source`, `target`) |
| `unlink` | array of strings | **supported** | Target paths to remove when safe (`~` ok) |

Import, generation, and activation **skip** Finder junk and bytecode: `.DS_Store`, AppleDouble `._*`, `__pycache__`, `.pyc` / `.pyo`, and `.git` (also `Thumbs.db`, `desktop.ini`, `.svn`). Those names are never copied into a generation or planned as file actions.

### `files.links`

Each entry:

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `source` | yes | string | Path relative to the directory containing `pourover.lua`, or absolute |
| `target` | yes | string | Path on disk (`~` expanded) |

**Behavior:**

- Plan/apply materializes **regular files** at `target` from generation blobs (content copied from `source` at build time). Directory sources expand to a tree of files under `target`.
- Editing files under `~/.pourover` is **not** live — run `pourover build` or `pourover apply` to refresh.
- Existing PourOver **symlinks** at the target are replaced with regular files on apply.
- If `target` is an unexpected type (e.g. directory) → plan error unless `policy.file_replace = "backup"` (or `"force"`), which moves the target aside under `state/backups/files/` before writing.
- Undeclared **owned** paths may be pruned when `policy.files_mode` is `safe` or `strict` (see below). Use `files.unlink` for explicit removals.
- Typical layout: sources under `~/.pourover/config/...`, targets under `~/.config/...`.

`pourover import --files` copies existing home/`~/.config` paths into the config tree and writes `files.links` (use `--force` if links are already declared). After apply, live paths are regular files activated from the generation, not symlinks.

### `files.managed`

Each entry:

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `source` | yes | string | Path relative to the config directory, or absolute |
| `target` | yes | string | Destination path on disk (`~` expanded) |

Copies `source` to `target` for apps that reject symlinks. Empty `source`/`target` (after trim) fail validation. Plan emits `managed_copy` when the target is missing or content differs; apply writes atomically. Regular files are overwritten in place. Unexpected target types (e.g. directories) fail unless `policy.file_replace = "backup"`, which moves the target aside then writes.

### `files.templates`

Each entry:

| Key | Required | Type | Description |
|-----|----------|------|-------------|
| `source` | yes | string | Path to a Go `text/template` file (relative to the config directory, or absolute) |
| `target` | yes | string | Destination path on disk (`~` expanded) |

Renders `source` with a fixed sandboxed context (no arbitrary code execution) **at generation build time**, then writes the result to `target` like managed copies. Empty `source`/`target` (after trim) fail validation. Plan emits `template_write` when the target is missing or content differs; JSON `value` is the generation **content hash**, not a unified diff. Apply writes the blob atomically.

Template context fields include:

| Field | Example | Description |
|-------|---------|-------------|
| `{{.Hostname}}` | `mymac` | Machine hostname |
| `{{.User}}` | `chase` | Current username |
| `{{.Home}}` | `/Users/chase` | Home directory |

`.Env` exists on the context struct for a future allowlist but is empty — do not rely on `{{index .Env "…"}}` for secrets or arbitrary environment variables. Rendering uses Go `text/template` with `missingkey=error` and no custom FuncMap (no shell-out helpers).

Example source (`config/gitconfig.tmpl`):

```text
[user]
	name = {{.User}}
	email = {{.User}}@{{.Hostname}}.local
```

```lua
files = {
  templates = {
    { source = "config/gitconfig.tmpl", target = "~/.gitconfig" },
  },
}
```

### `files.unlink`

Array of target path strings to remove when safe (`~` expanded). Each entry must be non-empty after trim. Plan emits `file_unlink` for existing symlinks or regular files (directories are refused). Apply removes them with the same safeguards.

Example:

```lua
files = {
  links = {
    { source = "config/nvim", target = "~/.config/nvim" },
  },
  managed = {
    { source = "config/foo.conf", target = "~/.config/foo.conf" },
  },
  templates = {
    { source = "config/gitconfig.tmpl", target = "~/.gitconfig" },
  },
  unlink = { "~/.old-dotfile" },
}
```

## `policy`

| Key | Required | Type | Default | Values |
|-----|----------|------|---------|--------|
| `uninstall_mode` | no | string | `"safe"` | `safe`, `strict`, `non_destructive` |
| `file_replace` | no | string | `"error"` | `error`, `backup` (`force` is an alias for `backup`) |
| `files_mode` | no | string | `"safe"` | `safe`, `strict`, `non_destructive` |

- **safe** — prompt before uninstalling undeclared Homebrew packages/taps (only formulae installed on request; dependency-only packages are ignored; formulae that are runtime deps of *declared* formulae are never removed, even if also installed on request; `homebrew/core` / `homebrew/cask` never untapped) and undeclared App Store apps when `packages.mas` is managed. A package listed under `formulae` or `casks` counts as declared for either type.
- **strict** — uninstall undeclared packages/taps (and managed MAS apps) without prompting
- **non_destructive** — never uninstall undeclared packages/taps (including MAS when managed)

### `policy.files_mode`

Controls pruning of **PourOver-owned** file targets that are no longer declared under `files.links`, `files.managed`, or `files.templates`. Ownership comes from `lock.json` `owned_files` (empty for old locks → no prune). Paths listed in `files.unlink` get `file_unlink` instead and are not also pruned.

- **safe** (default) — plan emits `file_prune`; apply prompts once before removing
- **strict** — plan emits `file_prune`; apply removes without prompting
- **non_destructive** — never emit prune actions

```lua
policy = {
  uninstall_mode = "safe",
  files_mode = "safe", -- or "strict", or "non_destructive"
}
```

### `policy.file_replace`

Controls what happens when a declared file target is an unexpected type (e.g. a directory blocking a regular-file write).

- **error** (default) — plan fails with a blocked-target error
- **backup** — move the existing target aside under the state directory, then write the file
- **force** — accepted synonym for `backup`

Backup destination:

```text
<stateDir>/backups/files/<UTC-timestamp>/<escaped-absolute-path>
```

Example: `~/Library/Application Support/PourOver/state/backups/files/20260815T053000Z/_Users_chase_.zshrc`

```lua
policy = {
  uninstall_mode = "safe",
  files_mode = "safe",
  file_replace = "backup", -- or "error" (default), or "force"
}
```

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

Declarative preferences via `defaults write` (nix-darwin `system.defaults`) plus optional security PAM. **Unset keys / omitted PAM tables are unmanaged.**

Searchable key list and Lua syntax: [`docs/macos-defaults.md`](macos-defaults.md).
Full nix-darwin option tree: [`docs/nix-darwin-options.md`](nix-darwin-options.md).

Generate from the live Mac: `pourover import macos` snapshots curated catalog keys into `macos.lua` (add-only merge by default; `--force` replaces curated sections; `--dry-run` previews). Expanding coverage means editing `internal/config/macos_catalog.yaml`, not scraping arbitrary domains. PAM is not imported yet.

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

`loginwindow` / `smb` / `SoftwareUpdate` write machine plists under `/Library/Preferences` via `sudo defaults` on apply. Wallpaper and Finder sidebar Favorites are not applied. Dock `persistent-apps` / `persistent-others` accept path arrays (nix-darwin-style tiles).

### `macos.security.pam.sudo_local`

Manages `/etc/pam.d/sudo_local` for Touch ID / Apple Watch / tmux reattach (nix-darwin `security.pam.sudo_local`). **Omitted `sudo_local` table = unmanaged** (no PAM plan actions).

| Key | Required | Type | Default | Description |
|-----|----------|------|---------|-------------|
| `enable` | no | boolean | `true` when the table is present | When `false`, write a managed stub (marker + disabled comment, no auth lines) instead of deleting — keeps `auth include sudo_local` safe |
| `reattach` | no | boolean | `false` | Emit `pam_reattach` line; auto-adds formula `pam-reattach` |
| `touch_id_auth` | no | boolean | `false` | Emit `auth sufficient pam_tid.so` |
| `watch_id_auth` | no | boolean | `false` | Emit `pam_watchid` line; resolves `pam_watchid.so` from common paths (not a Homebrew core formula — install manually, e.g. mostlygeek/pam-watchid) |

**Behavior:**

- Plan order includes brew first so implied PAM formulae (`pam-reattach` only) install before PAM file writes.
- Desired file starts with `# pourover: managed`, then lines in nix-darwin order: optional reattach → sufficient Touch ID → sufficient Watch ID.
- Ensures `auth include sudo_local` in `/etc/pam.d/sudo` when enabling.
- Writes under `/etc` need **admin** (`sudo` on apply). Pre-existing non-managed `sudo_local` is backed up then replaced when enabling.
- `enable = false` writes/replaces a PourOver-managed stub (or creates one if missing/empty); unmanaged non-empty files are left alone. Omitted table leaves any existing file alone.
- `watch_id_auth` fails plan/apply with a clear error if `pam_watchid.so` is not found under the Homebrew prefix `lib/pam` or `/opt/homebrew|/usr/local/lib/pam`.

```lua
macos = {
  security = {
    pam = {
      sudo_local = {
        enable = true, -- default when table present; set false to write disabled stub
        reattach = true,
        touch_id_auth = true,
        watch_id_auth = true,
      },
    },
  },
  defaults = {
    dock = { autohide = true },
  },
}
```

## Minimal example

See `test/fixtures/config/valid/pourover.lua`.

## Module layout (hybrid config)

Root `pourover.lua` may import modules from the same directory:

```lua
local packages = require("packages")

return {
  packages = packages,
  policy = { uninstall_mode = "safe", files_mode = "safe", file_replace = "error" },
}
```

`require("name")` resolves to `name.lua` next to the root config file.
