# PourOver

Declarative Homebrew and dotfile management for macOS.

PourOver loads `~/.pourover/pourover.lua`, plans the diff against your machine, and applies Homebrew packages plus symlink file links.

## Install

```bash
git clone https://github.com/chasebank87/PourOver.git
cd PourOver
make build    # or: go build -o pourover ./cmd/pourover
make test
```

Requires macOS, Go (to build), and Homebrew. CI runs `make vet`, `make test`, and `make build` on push/PR.

## Quick start

```bash
pourover init
pourover doctor
pourover plan
pourover apply --dry-run
pourover apply
```

`init` scaffolds `~/.pourover/` (`pourover.lua`, `packages.lua`, example `config/`). Use `--force` to overwrite.

`apply --dry-run` matches `plan`. Use `apply --yes` to skip uninstall confirmation prompts (CI).

## Commands

| Command | Purpose |
|---------|---------|
| `init` | Scaffold config |
| `plan` | Show pending actions (`--json` for machine-readable) |
| `apply` | Reconcile system (`--dry-run`, `--yes`) |
| `doctor` | Check brew, config, state dir, iCloud |
| `backup` | Force local snapshot (+ iCloud mirror when enabled) |
| `restore` | Restore `lock.json` / `last-plan.json` (`--snapshot`, `--icloud`) |

Global flags: `--config`, `--verbose` / `-v`, `--json`.

## Policies

`policy.uninstall_mode` in Lua:

- **safe** (default) — prompt once before uninstalling undeclared packages
- **strict** — uninstall without prompting
- **non_destructive** — never uninstall undeclared packages

## Paths

| Role | Default |
|------|---------|
| Config | `~/.pourover/` |
| State | `~/Library/Application Support/PourOver/state/` |
| iCloud mirror | `~/Library/Mobile Documents/com~apple~CloudDocs/PourOver/` |

State artifacts: `lock.json`, `last-plan.json`, `history/`, `snapshots/`.

Enable iCloud mirroring in config:

```lua
backup = { icloud = { enabled = true } }  -- optional: path = "..."
```

When enabled, successful `apply` and `backup` write a local snapshot and mirror it if iCloud Drive is available.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success or no-op |
| `1` | Runtime failure (config, plan, apply, doctor, backup/restore) |
| `2` | Invalid invocation (unknown command or bad flags) |

## Docs

- [Config schema](docs/config-schema.md)
- [Plan output format](docs/plan-output.md)
- [v2 backlog](docs/v2-backlog.md) — deferred features so v1 stays focused
- [v1 micro-steps](docs/plans/2026-05-18-pourover-v1-micro-steps.md)
