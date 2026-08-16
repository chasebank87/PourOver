# Plan output format

`pourover plan` prints the reconciliation plan. Use `--json` for machine-readable output.

On a TTY, text `plan` prints a PourOver header (`☕ PourOver  plan`) then the action list. `--json` and pipes stay plain (`plan.RenderText` / `plan.RenderJSON` with no header).

## JSON (`--json`)

Top-level object with an `actions` array. Each action:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | See types below |
| `name` | string | Homebrew token, MAS display name, file target path, PAM path, or `domain key` for defaults |
| `source` | string | Link, managed, or template source path (as declared in config) |
| `domain` | string | Apple defaults domain (`defaults_write` only) |
| `key` | string | Preference key (`defaults_write` only) |
| `value` | string | See per-type notes below |
| `kind` | string | `bool`, `int`, `float`, `string`, or `array` (`defaults_write`); `backup` for blocked managed/template targets when `file_replace` is backup |
| `trusted` | boolean | `tap_add` only: whether to `brew trust --tap` after tapping |

### `type` values

| `type` | Meaning |
|--------|---------|
| `tap_add` | Ensure tap is present |
| `tap_trust` | `brew trust --tap` for an already-tapped untrusted tap |
| `tap_remove` | Untap (follows `policy.uninstall_mode`) |
| `formula_install` / `formula_remove` / `formula_upgrade` | Homebrew formula |
| `cask_install` / `cask_remove` / `cask_upgrade` | Homebrew cask |
| `cask_rename` | Advisory: old token in `name`, new token in `value` |
| `mas_install` / `mas_remove` / `mas_upgrade` | Mac App Store app (`name` = display name, `value` = numeric ID) |
| `link_create` / `link_update` / `link_replace` | Declared `files.links` (text says create/update/replace **file**) |
| `managed_copy` | `files.managed` copy |
| `template_write` | `files.templates` write; `value` is a **content hash**, not a unified diff |
| `file_unlink` | Explicit `files.unlink` |
| `file_prune` | Owned path no longer declared |
| `defaults_write` | macOS `defaults`; `value` is the desired value as text |
| `pam_sudo_local_write` | Write `/etc/pam.d/sudo_local` (`value` = desired body or disabled stub) |
| `pam_sudo_local_remove` | Legacy: apply writes a disabled stub (does not delete) |
| `pam_sudo_include` | Ensure `auth include sudo_local` in `/etc/pam.d/sudo` |

Example:

```json
{
  "actions": [
    {
      "type": "formula_install",
      "name": "git"
    }
  ]
}
```

Field names are stable for tooling.

## Text (default)

One action per line:

```
tap oven-sh/bun
trust tap heroku/brew
untap unused/tap
upgrade formula git
install formula fzf
remove cask slack
cask renamed: windsurf → devin-desktop (update packages.lua)
install mas Xcode (497799835)
remove mas WhatsApp Messenger (310633997)
upgrade mas Keynote (409183694)
defaults write com.apple.dock autohide = true
pam write /etc/pam.d/sudo_local
pam disable stub /etc/pam.d/sudo_local
pam include sudo_local in /etc/pam.d/sudo
create file ~/.config/nvim <- config/nvim
update file ~/.zshrc <- config/home/zshrc
replace file ~/.zshrc <- config/zshrc (backup)
managed copy ~/.config/foo <- config/foo
template write ~/.config/foo <- config/foo.tmpl
unlink ~/.old-dotfile
prune file ~/.config/old
```

`cask_rename` is advisory only (apply does not mutate); replace the old token in config with the new name.

`link_replace` appears when `policy.file_replace` is `backup` (or `force`) and the target is an unexpected type (e.g. a directory); apply moves the target under `state/backups/files/` then writes a **regular file** from the generation blob. Text output is `replace file … (backup)`. Content drift and legacy PourOver symlinks are planned as `link_update` (`update file` in text) and written as regular files on apply. Missing targets are `link_create` (`create file`).

`managed_copy` and `file_unlink` appear in the plan when discovered; apply copies atomically and unlinks with directory safeguards. Managed copies against unexpected target types (e.g. directories) require `file_replace = "backup"` and are labeled `(backup)` in text output.

`template_write` appears when a `files.templates` target is missing or its content differs from the generation blob (rendered at build time). Text output is one line; JSON `value` holds the **content hash** used to read the blob. Apply writes the blob atomically like managed copies. Blocked template targets follow the same `file_replace` backup/error rules as managed copies.

`file_prune` appears for PourOver-owned paths (from `lock.json`) that are no longer declared under links/managed/templates when `policy.files_mode` is `safe` or `strict`. `non_destructive` never plans prune. Apply prompts once in `safe` (multiline `Proceed? [y/N]` list; skipped if declined), removes without prompting in `strict`, and soft-fails per path.

`pourover upgrade --dry-run` merges upgrade actions (outdated declared packages only) ahead of the normal apply plan.

Plan order: upgrade actions (upgrade command only), then brew/mas/pam/defaults, then generation file actions (links/managed/templates), then unlinks, then owned-file prunes.

When there is nothing to do:

```
No changes.
```
