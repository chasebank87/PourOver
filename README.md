# PourOver
Declarative Homebrew and dotfile management for macOS.

## Quick start

Preview changes without modifying your system (same output as `plan`):

```bash
pourover apply --dry-run
pourover plan
```

Apply packages and file links (policy-aware brew removes; use --yes in CI):

```bash
pourover apply
pourover apply --yes
pourover plan   # should show no changes after a full apply
```

## Docs

- [Config schema](docs/config-schema.md)
- [Plan output format](docs/plan-output.md)
- [v2 backlog](docs/v2-backlog.md) — deferred features so v1 stays focused
