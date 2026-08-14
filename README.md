# PourOver
Declarative Homebrew and dotfile management for macOS.

## Quick start

Preview changes without modifying your system (same output as `plan`):

```bash
pourover apply --dry-run
pourover plan
```

Apply Homebrew packages (installs and policy-aware removes). File links are planned but not applied yet:

```bash
pourover apply
pourover apply --yes   # skip uninstall prompts (CI)
pourover plan          # should show no brew actions after a full apply
```

## Docs

- [Config schema](docs/config-schema.md)
- [Plan output format](docs/plan-output.md)
- [v2 backlog](docs/v2-backlog.md) — deferred features so v1 stays focused
