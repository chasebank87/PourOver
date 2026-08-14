# PourOver

Declarative Homebrew and dotfile management for macOS.

PourOver loads `~/.pourover/pourover.lua`, plans the diff against your machine, and applies Homebrew packages plus symlink file links.

## Install

One-liner (downloads the latest GitHub Release binary; installs Homebrew first if missing):

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/chasebank87/PourOver/main/scripts/install.sh)"
```

From source:

```bash
git clone https://github.com/chasebank87/PourOver.git
cd PourOver
make build    # or: go build -o pourover ./cmd/pourover
make install  # copies to ~/.local/bin by default (PREFIX=/usr/local make install)
make test
```

Requires macOS. The installer bootstraps Homebrew when it is not already present. Building from source also needs Go. CI runs `make vet`, `make test`, and `make build` on push/PR; tagged `v*` releases publish darwin archives via GoReleaser.

## Quick start

New machine:

```bash
pourover init
pourover doctor
pourover plan
pourover apply --dry-run
pourover apply
```

Existing brew packages / configs already on the Mac:

```bash
pourover init          # if needed
pourover import        # seed packages.lua + file links (use --force to overwrite)
pourover plan
pourover apply
```

`init` scaffolds `~/.pourover/` (`pourover.lua`, `packages.lua`, example `config/`). Use `--force` to overwrite.

`apply --dry-run` matches `plan`. Use `apply --yes` to skip uninstall confirmation prompts (CI).

Keep packages (and PourOver itself) up to date — similar to `brew update` + `brew upgrade`:

```bash
pourover self-update       # update the pourover binary from GitHub Releases
pourover upgrade           # self-update, brew upgrade declared packages, then reapply
pourover upgrade --dry-run # preview package/apply actions (skips self-update)
pourover upgrade --skip-self-update
```

## Commands

| Command | Purpose |
|---------|---------|
| `init` | Scaffold config |
| `import` | Import installed brew packages and common files into config |
| `plan` | Show pending actions (`--json` for machine-readable) |
| `apply` | Reconcile system (`--dry-run`, `--yes`) |
| `upgrade` | Self-update pourover, upgrade declared packages, then reapply (`--dry-run`, `--yes`, `--skip-self-update`) |
| `self-update` | Replace the pourover binary from the latest GitHub Release |
| `doctor` | Check PATH, brew, config, state dir, iCloud |
| `backup` | Force local snapshot (+ iCloud mirror when enabled) |
| `restore` | Restore `lock.json` / `last-plan.json` (`--snapshot`, `--icloud`) |

Global flags: `--config`, `--verbose` / `-v`, `--json`.

### Import flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--packages` | on | Write `packages.lua` from `brew list` |
| `--files` | on | Import `~/.config/*` and common home dotfiles into `files.links` |
| `--dry-run` | off | Preview only |
| `--force` | off | Overwrite non-empty packages/links sections |

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
