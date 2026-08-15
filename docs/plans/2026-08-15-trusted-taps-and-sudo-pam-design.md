# Trusted taps + sudo PAM design

**Status:** Approved  
**Date:** 2026-08-15

## Vision

Declarative parity with nix-darwin for:

1. **Trusted Homebrew taps** — opt-out `trusted` on tap entries (plain strings keep today’s auto-trust).
2. **`sudo_local` PAM** — Touch ID / Apple Watch / pam_reattach for sudo via `/etc/pam.d/sudo_local`.

## Decisions

| Topic | Choice |
|-------|--------|
| Scope | Both features |
| Tap trust default | Opt-out: string or table defaults `trusted = true`; `trusted = false` skips `brew trust` |
| PAM formulae | Auto-require `pam-reattach` when reattach enabled; resolve `pam_watchid.so` via path search (not brew core) |
| Disable semantics | Omitted `sudo_local` = unmanaged; `enable = false` writes managed stub (keeps include safe) |
| Approach | Dedicated domains (tap schema + security reconciler), not `files.managed` hacks |

## Non-goals

- `system.primaryUser`
- nix `environment.systemPackages` (use `packages.formulae`)
- Arbitrary PAM services beyond `sudo_local`
- TUI toggles (CLI/plan/apply first)

## Config shape

```lua
packages = {
  taps = {
    "homebrew/cask-fonts",
    { name = "oven-sh/bun", trusted = true },
    { name = "heroku/brew", trusted = false },
  },
}

macos = {
  security = {
    pam = {
      sudo_local = {
        enable = true,
        reattach = true,
        touch_id_auth = true,
        watch_id_auth = true,
      },
    },
  },
  defaults = { ... },
}
```

## Architecture

### Taps

- Decode taps as string **or** `{ name, trusted? }` → `[]TapSpec{Name, Trusted}`.
- `Trusted` defaults to `true`.
- Plan/`AddTap`: call `brew trust --tap` only when `Trusted && NeedsExplicitTrust(name)`.
- Import continues to emit plain strings (implicit trusted).

### PAM

```text
macos.security.pam.sudo_local
  → ensure formula pam-reattach in brew plan when reattach; resolve pam_watchid.so from common paths when watch_id_auth
  → desired /etc/pam.d/sudo_local text
  → ensure `auth include sudo_local` in /etc/pam.d/sudo
  → apply after brew installs; admin write
  → enable=false → write managed stub (keeps include safe)
```

Generated lines (nix-darwin order):

1. `auth optional <prefix>/lib/pam/pam_reattach.so` if `reattach`
2. `auth sufficient pam_tid.so` if `touch_id_auth`
3. `auth sufficient <prefix>/lib/pam/pam_watchid.so` if `watch_id_auth`

File starts with `# pourover: managed` marker.

## Edge cases

| Case | Behavior |
|------|----------|
| Plain tap string | Trusted=true |
| `trusted=false` third-party | Tap only; no trust / no tap_trust drift |
| Official homebrew/* | Trust no-op |
| `sudo_local` omitted | No PAM management |
| `enable=false` | Replace with managed stub if PourOver-managed/empty; leave unmanaged alone |
| Pre-existing sudo_local | Backup then replace when enabling |
| Missing .so | Fail apply clearly |
| sudo missing include | Add include line only |

## Testing

- Lua decode: string + table taps; PAM flags
- Brew plan: trust gating
- PAM text generation; enable=false removal
- Stub filesystem for `/etc/pam.d`; no live sudo in CI
