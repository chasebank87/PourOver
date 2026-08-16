# nix-darwin options → PourOver

Searchable coverage of every top-level option-set on [MyNixOS nix-darwin options](https://mynixos.com/nix-darwin/options).

PourOver is **not** a Nix module system. This page tells you what maps today, what is a macOS `defaults` key (see [macos-defaults.md](macos-defaults.md)), and what is indexed only.

Status:

- **supported** — PourOver applies it
- **mapped** — same intent via an existing PourOver domain
- **index-only** — documented here; not applied

## Top-level option-sets (17)

| MyNixOS | What it is | Status | PourOver |
|---------|------------|--------|----------|
| [`_module`](https://mynixos.com/nix-darwin/options/_module) | Nix module internals | index-only | — |
| [`lib`](https://mynixos.com/nix-darwin/option/lib) | Helper functions | index-only | — |
| [`documentation`](https://mynixos.com/nix-darwin/options/documentation) | Install man/info/doc from nix packages | index-only | Nix-only |
| [`environment`](https://mynixos.com/nix-darwin/options/environment) | `/etc`, PATH, launch agents, system packages | mapped in part | `files.links` for user files; nix store PATH is Nix-only |
| [`fonts`](https://mynixos.com/nix-darwin/options/fonts) | Fonts into `/Library/Fonts/Nix Fonts` | index-only | Install font casks via `packages.casks` |
| [`homebrew`](https://mynixos.com/nix-darwin/options/homebrew) | Formulae, casks, taps, mas, brew bundle | mapped | `packages.formulae`, `packages.casks`, `packages.taps` (string or `{ name, trusted? }`), `packages.mas`; cargo/go/vscode not yet |
| [`launchd`](https://mynixos.com/nix-darwin/options/launchd) | Agents and daemons | index-only | — |
| [`networking`](https://mynixos.com/nix-darwin/options/networking) | Hostname, DNS, computer name | index-only | Often `scutil` / System Settings, not `defaults` |
| [`nix`](https://mynixos.com/nix-darwin/options/nix) | Nix daemon and nix.conf | index-only | PourOver is not Nix |
| [`nixpkgs`](https://mynixos.com/nix-darwin/options/nixpkgs) | nixpkgs config | index-only | — |
| [`power`](https://mynixos.com/nix-darwin/options/power) | Sleep, restart after freeze/power loss | index-only | Typically `pmset` |
| [`programs`](https://mynixos.com/nix-darwin/options/programs) | zsh, tmux, vim, direnv, 1Password, … | index-only | Install with brew; configure with `files.links` |
| [`security`](https://mynixos.com/nix-darwin/options/security) | PAM, sudo, PKI, sandbox | mapped in part | `macos.security.pam.sudo_local` (Touch ID / Watch / reattach); other security options index-only |
| [`services`](https://mynixos.com/nix-darwin/options/services) | yabai, skhd, postgres, tailscale, … (42) | index-only | Install the app/cask; no launchd module yet |
| [`system`](https://mynixos.com/nix-darwin/options/system) | stateVersion, primaryUser, activation, **defaults** | mixed | See `system.defaults` below |
| [`time`](https://mynixos.com/nix-darwin/options/time) | Time zone | index-only | `systemsetup` / System Settings |
| [`users`](https://mynixos.com/nix-darwin/options/users) | nix-darwin user/group accounts | index-only | — |

## `homebrew` (mapped)

| nix-darwin | PourOver |
|------------|----------|
| `homebrew.brews` | `packages.formulae` |
| `homebrew.casks` | `packages.casks` |
| `homebrew.taps` | `packages.taps` — string or `{ name, trusted? }` (`trusted` defaults `true`; `false` skips `brew trust`) |
| `homebrew.masApps` | `packages.mas` (name → numeric ID; omit = unmanaged; `{}` = manage zero apps) |
| `homebrew.enable` / brew bundle flags | installer bootstraps Homebrew; `pourover apply` runs `brew` |

## `system.defaults` (supported)

Leaf keys and Lua syntax: **[macos-defaults.md](macos-defaults.md)**.

| nix-darwin section | Status | Notes |
|--------------------|--------|--------|
| [dock](https://mynixos.com/nix-darwin/options/system.defaults.dock) | supported | includes `persistent-apps` / `persistent-others` (path arrays → dock tiles) |
| [finder](https://mynixos.com/nix-darwin/options/system.defaults.finder) | supported | |
| [NSGlobalDomain](https://mynixos.com/nix-darwin/options/system.defaults.NSGlobalDomain) | supported | some keys need logout |
| [.GlobalPreferences](https://mynixos.com/nix-darwin/options/system.defaults.%22.GlobalPreferences%22) | supported | Lua: `macos.defaults[".GlobalPreferences"]` |
| [trackpad](https://mynixos.com/nix-darwin/options/system.defaults.trackpad) | supported | also writes Bluetooth trackpad domain |
| [magicmouse](https://mynixos.com/nix-darwin/options/system.defaults.magicmouse) | supported | also writes Bluetooth mouse domain |
| [screencapture](https://mynixos.com/nix-darwin/options/system.defaults.screencapture) | supported | |
| [screensaver](https://mynixos.com/nix-darwin/options/system.defaults.screensaver) | supported | |
| [spaces](https://mynixos.com/nix-darwin/options/system.defaults.spaces) | supported | |
| [menuExtraClock](https://mynixos.com/nix-darwin/options/system.defaults.menuExtraClock) | supported | |
| [hitoolbox](https://mynixos.com/nix-darwin/options/system.defaults.hitoolbox) | supported | Fn key stored as int |
| [iCal](https://mynixos.com/nix-darwin/options/system.defaults.iCal) | supported | |
| [LaunchServices](https://mynixos.com/nix-darwin/options/system.defaults.LaunchServices) | supported | |
| [ActivityMonitor](https://mynixos.com/nix-darwin/options/system.defaults.ActivityMonitor) | supported | |
| [WindowManager](https://mynixos.com/nix-darwin/options/system.defaults.WindowManager) | supported | Stage Manager / tiling |
| [universalaccess](https://mynixos.com/nix-darwin/options/system.defaults.universalaccess) | supported | |
| [controlcenter](https://mynixos.com/nix-darwin/options/system.defaults.controlcenter) | supported (best-effort) | nix-darwin uses ByHost plists |
| [loginwindow](https://mynixos.com/nix-darwin/options/system.defaults.loginwindow) | supported (system) | `/Library/Preferences`; needs admin |
| [smb](https://mynixos.com/nix-darwin/options/system.defaults.smb) | supported (system) | needs admin |
| [SoftwareUpdate](https://mynixos.com/nix-darwin/options/system.defaults.SoftwareUpdate) | supported (system) | needs admin |
| [CustomUserPreferences](https://mynixos.com/nix-darwin/option/system.defaults.CustomUserPreferences) | supported | `macos.defaults.custom` |
| [CustomSystemPreferences](https://mynixos.com/nix-darwin/option/system.defaults.CustomSystemPreferences) | index-only | use `custom` only for user domains; system paths need sudo |

## `services` (index-only)

[MyNixOS services](https://mynixos.com/nix-darwin/options/services): aerospace, autossh, buildkite-agents, cachix-agent, chunkwm, dnscrypt-proxy, dnsmasq, emacs, eternal-terminal, github-runners, gitlab-runner, hercules-ci-agent, ipfs, jankyborders, karabiner-elements, khd, kwm, lorri, mopidy, netbird, netdata, nextdns, nix-daemon, ofborg, offlineimap, openssh, postgresql, privoxy, prometheus, redis, sketchybar, skhd, spacebar, spotifyd, synapse-bt, synergy, tailscale, telegraf, trezord, yabai.

Install the corresponding brew formula/cask in `packages`; PourOver does not generate launchd plists yet.

## `programs` (index-only)

[MyNixOS programs](https://mynixos.com/nix-darwin/options/programs): 1password, 1password-gui, arqbackup, bash, direnv, fish, gnupg, info, man, nix-index, ssh, tmux, vim, zsh.

## `environment` children (index-only except files)

[MyNixOS environment](https://mynixos.com/nix-darwin/options/environment): systemPackages, paths, variables, shells, loginShellInit, `etc`, LaunchAgents/Daemons. User dotfiles → `files.links`.

## `security` (mapped in part)

| nix-darwin | PourOver |
|------------|----------|
| `security.pam.sudo_local` | `macos.security.pam.sudo_local` — manages `/etc/pam.d/sudo_local` + sudo include; auto formula `pam-reattach`; `pam-watchid` resolved from common paths (manual install); admin on apply |
| Other `security.*` (PKI, sandbox, …) | index-only |

See [config-schema.md](config-schema.md) for Lua flags (`enable`, `reattach`, `touch_id_auth`, `watch_id_auth`).

## `networking` / `power` / `time` / `users` / `launchd`

See the top-level table. These are not `defaults write` catalogs; they stay index-only until PourOver grows matching apply domains.
