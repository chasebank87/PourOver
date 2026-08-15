# Plan output format (v1)

`pourover plan` prints the reconciliation plan. Use `--json` for machine-readable output.

## JSON (`--json`)

Top-level object with an `actions` array. Each action:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `formula_install`, `formula_remove`, `formula_upgrade`, `cask_install`, `cask_remove`, `cask_upgrade`, `cask_rename`, `link_create`, `link_update`, `link_replace`, `managed_copy`, `file_unlink`, `file_prune`, `defaults_write` |
| `name` | string | Homebrew package name, link/managed/unlink target path, or `domain key` for defaults |
| `source` | string | Link or managed source path (as declared in config) |
| `domain` | string | Apple defaults domain (`defaults_write` only) |
| `key` | string | Preference key (`defaults_write` only) |
| `value` | string | Desired value rendered as text (`defaults_write` only); for `cask_rename`, the new Homebrew token |
| `kind` | string | `bool`, `int`, `float`, or `string` (`defaults_write` only) |

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

Field names are stable for future dashboard integration.

## Text (default)

One action per line: `<verb> <kind> <name>`

```
upgrade formula git
install formula fzf
remove cask slack
cask renamed: windsurf →devin-desktop (update packages.lua)
defaults write com.apple.dock autohide = true
link create ~/.config/nvim <- config/nvim
link replace ~/.zshrc <- config/zshrc (backup)
managed copy ~/.config/foo <- config/foo
unlink ~/.old-dotfile
prune file ~/.config/old
```

`cask_rename` is advisory only (apply does not mutate); replace the old token in config with the new name.

`link_replace` appears when `policy.file_replace` is `backup` (or `force`) and the link target exists as a non-symlink; apply moves the target under `state/backups/files/` then creates the link.

`managed_copy` and `file_unlink` appear in the plan when discovered; apply copies atomically and unlinks with directory safeguards. Managed copies against unexpected target types (e.g. directories) require `file_replace = "backup"` and are labeled `(backup)` in text output.

`file_prune` appears for PourOver-owned paths (from `lock.json`) that are no longer declared under links/managed when `policy.files_mode` is `safe` or `strict`. `non_destructive` never plans prune. Apply prompts once in `safe` (skipped if declined), removes without prompting in `strict`, and soft-fails per path.

`pourover upgrade --dry-run` merges upgrade actions (outdated declared packages only) ahead of the normal apply plan.

Plan order: upgrade actions (upgrade command only), then brew install/remove, then macOS `defaults_write`, then file link actions, then managed copies, then unlinks, then owned-file prunes.

When there is nothing to do:

```
No changes.
```
