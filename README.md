# PourOver

Declarative Homebrew, Mac App Store, defaults, and dotfile management for macOS.

**macOS only.** Linux and other platforms are out of scope.

## Install

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/chasebank87/PourOver/main/scripts/install.sh)"
```

That downloads the latest GitHub Release, bootstraps [Homebrew](https://brew.sh) if it is missing, and installs `pourover` plus a `pour` alias (same CLI). A published `v*` release must exist.

If the install directory is not on `PATH` (often `~/.local/bin`), add it:

```bash
export PATH="$HOME/.local/bin:$PATH"
eval "$(brew shellenv)"   # if brew was just installed
```

Then: `pour init` (or `pourover init`).

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
pourover import macos  # snapshot curated defaults into macos.lua
pourover plan
pourover apply
```

`init` scaffolds `~/.pourover/` (`pourover.lua`, `packages.lua`, example `config/`). Use `--force` to overwrite. `apply --dry-run` matches `plan`. Use `apply --yes` to skip confirmation prompts (CI).

## What apply does

`pourover apply` evaluates Lua, writes an **activation generation** under Application Support (frozen packages + file blobs), then activates it onto the live system. Editing sources under `~/.pourover` is **not** live — run `apply` (or `build`) to refresh.

Declared `files.links` are copied to targets as **regular files** (legacy PourOver symlinks are replaced). JSON action types stay `link_create` / `link_update` / `link_replace`; plan text says `create file` / `update file` / `replace file`. Finder junk (`.DS_Store`, AppleDouble `._*`), `__pycache__` / `.pyc`, and `.git` directories are skipped during import and activation.

In `policy.files_mode = "safe"` (default), apply lists owned-but-undeclared files one per line and asks `Proceed? [y/N]` before prune. `--yes` skips that prompt.

This is evaluate-then-activate (nix-darwin inspired), not a Nix store. Homebrew remains the package engine.

## TUI

On an interactive terminal (stdin and stdout are TTYs), `pourover` with **no arguments** opens the TUI. `pourover tui` always opens it. Subcommands stay CLI.

Auto-launch does **not** run when a subcommand or flags are passed, when stdin/stdout are not a TTY, or when `CI=true`.

Home screens: Plan, Apply, Upgrade, Doctor (opt-in fixes), History, Backup/Restore, Import, Config (iCloud + git), Self-update. **History is TUI-only** (no `pourover history` command). The home `drift:` line counts pending plan actions; Plan shows the list.

## Commands

| Command | Purpose |
|---------|---------|
| `tui` | Open the interactive TUI (also auto-launches on interactive no-args) |
| `init` | Scaffold config |
| `import` | Import installed brew packages, App Store apps, and common files (`import macos` for defaults → `macos.lua`) |
| `config` | iCloud mirror and git config sync (`config icloud`, `config git`, `push` / `pull`) |
| `build` | Freeze config into an activation generation (no live writes) |
| `plan` | Show pending actions (`--json` for machine-readable) |
| `apply` | Reconcile the system (`--dry-run`, `--yes`, `--quiet`) |
| `upgrade` | Self-update pourover, upgrade **outdated** declared brew/mas packages, then reapply (`--dry-run`, `--yes`, `--skip-self-update`, `--quiet`) |
| `self-update` | Replace the pourover binary from the latest GitHub Release |
| `doctor` | Check PATH, brew, config, state dir, iCloud, git sync |
| `backup` | Force local snapshot (+ iCloud mirror when enabled) |
| `restore` | Restore `lock.json` / `last-plan.json` (`--snapshot`, `--icloud`) |

Global flags: `--config`, `--verbose` / `-v`, `--json`.

`plan` / `apply` / `upgrade` show a PourOver header on a TTY (`NO_COLOR` / pipes / `--json` stay plain). Apply and upgrade use a progress bar; Homebrew logs scroll underneath. `--quiet` / `-q` is summary-only.

### Import flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--packages` | on | Merge installed brew taps/formulae/casks and App Store apps (`mas list`) into `packages.lua` |
| `--files` | on | Merge common home/`~/.config` paths into `files.links` |
| `--dry-run` | off | Preview only |
| `--force` | off | Replace packages/links with the discovered set (default is add-only merge) |

Re-import without `--force` only adds newly discovered items. Core taps (`homebrew/core`, `homebrew/cask`) are not written. App Store import writes `packages.mas` when `mas` is available.

`pourover import macos` is a separate subcommand (not `--macos`). It snapshots readable curated catalog keys into `macos.lua` and wires `require("macos")`. Same `--dry-run` / `--force` semantics.

## Config sketch

Taps may be strings or `{ name, trusted = false }` (default `trusted = true`). Mac App Store apps are a name → numeric ID map. Omit `mas` to leave App Store unmanaged; `mas = {}` manages and desires zero apps. Declaring `mas` implies the `mas` Homebrew formula. Apply runs `mas install`, then `mas get` on failure (first-time apps). Sign in to the App Store in the GUI.

```lua
-- packages.lua
return {
  formulae = { "git" },
  casks = { "raycast" },
  mas = {
    Xcode = 497799835,
    ["1Password for Safari"] = 1569813296,
  },
}
```

macOS preferences (nix-darwin-style `defaults`) are **not** in the default init config. Snapshot them:

```bash
pourover import macos            # merge into macos.lua (add-only)
pourover import macos --force    # replace curated sections
pourover import macos --dry-run
```

```lua
macos = {
  defaults = {
    dock = { autohide = true, ["show-recents"] = false, tilesize = 48 },
    finder = { ShowPathbar = true, AppleShowAllExtensions = true },
    screencapture = { type = "png" },
  },
}
```

System-scope keys (`loginwindow`, `smb`, `SoftwareUpdate`) write under `/Library/Preferences` via `sudo defaults`. Expand coverage in `internal/config/macos_catalog.yaml`.

Optional sudo Touch ID / Apple Watch (writes `/etc/pam.d`; needs admin on apply):

```lua
macos = {
  security = {
    pam = {
      sudo_local = {
        enable = true,
        reattach = true,
        touch_id_auth = true,
        watch_id_auth = false,
      },
    },
  },
}
```

Keep packages (and PourOver itself) up to date — brew and mas for **outdated declared** packages only:

```bash
pourover self-update
pourover upgrade
pourover upgrade --dry-run          # skips self-update
pourover upgrade --skip-self-update
```

`apply` only installs missing packages (`HOMEBREW_NO_INSTALL_UPGRADE`). Retired cask tokens show `cask renamed: …` — update `packages.lua`; apply will not keep re-installing the old name. Auto-updating casks that are already current on disk are skipped; use `brew upgrade --cask --greedy` for those.

## Policies

`policy.uninstall_mode`:

- **safe** (default) — prompt once before uninstalling undeclared on-request packages/taps (dependency-only formulae ignored; core taps never untapped), and undeclared App Store apps when `packages.mas` is managed
- **strict** — uninstall without prompting
- **non_destructive** — never uninstall undeclared packages (including MAS when managed)

`policy.file_replace`:

- **error** (default) — plan fails when a declared file target is an unexpected type (e.g. a directory)
- **backup** — move the target aside under `<stateDir>/backups/files/<timestamp>/…`, then write (`force` is an alias)

`policy.files_mode` (PourOver-owned undeclared targets from `lock.json` `owned_files`):

- **safe** (default) — plan emits `file_prune`; apply prompts once (multiline list)
- **strict** — prune without prompting
- **non_destructive** — never plan or apply prune

Use `files.unlink` for explicit removals. Old locks with empty `owned_files` never invent prune candidates.

Packages listed under `formulae` or `casks` count as declared for either type. Prefer GUI apps in `casks`. App Store apps are `packages.mas` by ID, not casks.

## Paths

| Role | Default |
|------|---------|
| Config | `~/.pourover/` |
| State | `~/Library/Application Support/PourOver/state/` |
| iCloud mirror | `~/Library/Mobile Documents/com~apple~CloudDocs/PourOver/` |

State artifacts: `lock.json` (`owned_files`, `generation_id`), `current`, `generations/<id>/` (manifest + blobs), `last-plan.json`, `history/`, `snapshots/`, `backups/files/`.

iCloud state mirroring:

```bash
pourover config icloud enable          # optional: --path /custom/dir
# or: backup = { icloud = { enabled = true } }
```

Successful `apply` and `backup` write a local snapshot and mirror it when iCloud Drive is available.

### GitHub config backup

Keep `~/.pourover` itself in a git repo:

```bash
pourover config git setup git@github.com:USER/pourover-config.git

# new machine
pourover config git restore git@github.com:USER/pourover-config.git
pourover apply

pourover config push
pourover config pull
```

With `backup.git.enabled` and `auto_push = true`, successful `apply` / `import` commit and push when the config tree is dirty (warnings only; they never fail apply).

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success or no-op |
| `1` | Runtime failure (config, plan, apply, doctor, backup/restore) |
| `2` | Invalid invocation (unknown command or bad flags) |

## Docs

- [Config schema](docs/config-schema.md)
- [Plan output format](docs/plan-output.md)
- [macOS defaults catalog](docs/macos-defaults.md)
- [nix-darwin options map](docs/nix-darwin-options.md)

## Contributing

From source (needs Go):

```bash
git clone https://github.com/chasebank87/PourOver.git
cd PourOver
make build    # or: go build -o pourover ./cmd/pourover
make install  # ~/.local/bin by default (PREFIX=/usr/local make install)
make test
```

CI runs `make vet`, `make test`, and `make build` on push/PR. Tagged `v*` releases publish darwin archives via GoReleaser (`self-update` and the install script use those).

Publish a release:

1. **Tag push** — `git tag v0.3.3 && git push origin v0.3.3`
2. **Actions UI** — Actions → **release** → Run workflow → enter `0.3.3` or `v0.3.3` (tags current `main` and runs GoReleaser; token tag pushes do not start a second run)
