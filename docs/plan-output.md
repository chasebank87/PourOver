# Plan output format (v1)

`pourover plan` prints the reconciliation plan. Use `--json` for machine-readable output.

## JSON (`--json`)

Top-level object with an `actions` array. Each action:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `formula_install`, `formula_remove`, `formula_upgrade`, `cask_install`, `cask_remove`, `cask_upgrade`, `cask_rename`, `link_create`, `link_update`, `managed_copy`, `file_unlink`, `defaults_write` |
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
managed copy ~/.config/foo <- config/foo
unlink ~/.old-dotfile
```

`cask_rename` is advisory only (apply does not mutate); replace the old token in config with the new name.

`managed_copy` and `file_unlink` appear in the plan when discovered; apply copies atomically and unlinks with directory safeguards.

`pourover upgrade --dry-run` merges upgrade actions (outdated declared packages only) ahead of the normal apply plan.

Plan order: upgrade actions (upgrade command only), then brew install/remove, then macOS `defaults_write`, then file link actions, then managed copies, then unlinks.

When there is nothing to do:

```
No changes.
```
