# Plan output format (v1)

`pourover plan` prints the reconciliation plan. Use `--json` for machine-readable output.

## JSON (`--json`)

Top-level object with an `actions` array. Each action:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `formula_install`, `formula_remove`, `formula_upgrade`, `cask_install`, `cask_remove`, `cask_upgrade`, `link_create`, `link_update`, `defaults_write` |
| `name` | string | Homebrew package name, link target path, or `domain key` for defaults |
| `source` | string | Link source path (links only, as declared in config) |
| `domain` | string | Apple defaults domain (`defaults_write` only) |
| `key` | string | Preference key (`defaults_write` only) |
| `value` | string | Desired value rendered as text (`defaults_write` only) |
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
defaults write com.apple.dock autohide = true
link create ~/.config/nvim <- config/nvim
```

`pourover upgrade --dry-run` merges upgrade actions (outdated declared packages only) ahead of the normal apply plan.

Plan order: upgrade actions (upgrade command only), then brew install/remove, then macOS `defaults_write`, then file link actions.

When there is nothing to do:

```
No changes.
```
