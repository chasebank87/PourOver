# Plan output format (v1)

`pourover plan` prints the reconciliation plan. Use `--json` for machine-readable output.

## JSON (`--json`)

Top-level object with an `actions` array. Each action:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `formula_install`, `formula_remove`, `formula_upgrade`, `cask_install`, `cask_remove`, `cask_upgrade`, `link_create`, `link_update` |
| `name` | string | Homebrew package name or link target path (as declared in config) |
| `source` | string | Link source path (links only, as declared in config) |

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
link create ~/.config/nvim <- config/nvim
```

`pourover upgrade --dry-run` merges upgrade actions ahead of the normal apply plan.

Plan order: upgrade actions (upgrade command only), then all brew install/remove actions, then all file link actions.

When there is nothing to do:

```
No changes.
```
