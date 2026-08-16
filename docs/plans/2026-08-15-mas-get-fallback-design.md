# MAS install → get fallback design

**Status:** Approved  
**Date:** 2026-08-15

## Vision

Match current Homebrew Bundle MAS behavior so first-time App Store apps on a signed-in Apple Account can be acquired without a manual GUI install first. When the App Store session is missing, fail with sign-in guidance instead of a raw “redownload unavailable” error.

## Problem

`mas install <id>` only **redownloads** apps already on the signed-in Apple Account. A fresh account (or an unauthenticated App Store session) fails with “Redownload Unavailable with this Apple Account” (or similar). Opening the App Store and getting the app once authenticates Media & Purchases and records the app as gotten; then `mas install` works.

`mas account` / `mas signin` are gone on current `mas` (v5+). There is no reliable CLI preflight for “signed in.”

## Decisions

| Topic | Choice |
|-------|--------|
| Apply path | `mas install <id>`; on **any** install failure, `mas get <id>` (same as current `brew bundle`) |
| Plan action | Unchanged: `mas_install` (text still `install mas Name (id)`) |
| Privilege | Still run `mas` as the GUI user (no `sudo mas`). `mas` 7 may prompt for admin itself |
| Upgrade | Unchanged: `mas upgrade <id>` only (app is already installed) |
| Old `mas` without `get` | Do not treat “unknown command” as success; keep the install error plus sign-in guidance |
| CI / `--yes` | Same install→get sequence; if both fail, error (no interactive wait) |

nix-darwin does not call `mas` itself; `homebrew.masApps` goes through `brew bundle`. Bundle’s current installer is `mas install` then `mas get` ([Homebrew/brew#21590](https://github.com/Homebrew/brew/pull/21590), [Homebrew/brew#22361](https://github.com/Homebrew/brew/pull/22361)).

## Non-goals

- Detecting Apple ID sign-in before apply (`mas account` is gone; no private StoreKit probe)
- Pausing apply to wait for Enter / auto-opening App Store.app
- Changing `mas uninstall` / `mas upgrade`
- Wrapping `mas` in `sudo` (that is the nix-darwin `installd` / root-session failure mode)

## Apply behavior

For each `mas_install` action:

1. `mas install <id>`
2. If that fails, `mas get <id>` (alias `purchase`; PourOver calls `get`)
3. If `get` succeeds, the action succeeds (count as installed)
4. If `get` is not a valid subcommand, return the **install** error plus guidance
5. If both fail, return the `get` error (or joined errors) plus guidance:

   Sign in to the App Store (Media & Purchases), then retry. First-time apps must be gotten onto this Apple Account (`mas get` opens Apple’s dialogs when a GUI session exists). Open the app page with `mas open <id>` if needed.

`mas get` is how free apps are first-acquired. Paid apps that were never purchased still fail; the same guidance applies (buy/get once in App Store).

## Docs

README / config-schema: `mas install` redownloads; apply falls back to `mas get` for first-time gets. You must be signed into the App Store in the GUI session.

## Testing

Fake `MasRunner` only (no live App Store):

- Install succeeds → `get` is not called
- Install fails, `get` succeeds → success; calls are `install` then `get`
- Both fail → error includes sign-in / App Store guidance
- `get` unknown command → original install error, no false success
- `ApplyMasInstalls` still continues after a per-app failure
