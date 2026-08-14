# PourOver
Declarative Homebrew and dotfile management for macOS.

## Quick start

```bash
pourover init
pourover plan
pourover apply --dry-run
pourover apply
```

`init` scaffolds `~/.pourover/` (`pourover.lua`, `packages.lua`, example `config/`). Use `--force` to overwrite.

Preview without modifying the system (`apply --dry-run` matches `plan`):

```bash
pourover apply --dry-run
pourover plan
```

Apply packages and file links (policy-aware brew removes; use `--yes` in CI):

```bash
pourover apply
pourover apply --yes
pourover plan   # should show no changes after a full apply
```

Successful applies write `lock.json` and `last-plan.json` under `~/Library/Application Support/PourOver/state/`.

## Docs

- [Config schema](docs/config-schema.md)
- [Plan output format](docs/plan-output.md)
- [v2 backlog](docs/v2-backlog.md) — deferred features so v1 stays focused
