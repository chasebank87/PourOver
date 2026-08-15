# Activation generation design

**Status:** Approved  
**Date:** 2026-08-15  
**Release:** v0.3.0

## Vision

Freeze evaluated Lua + declared file payloads into an **activation generation**, then activate that generation onto the live system. Editing `~/.pourover` is not live: `files.links` no longer create symlinks; apply writes regular files from generation blobs.

Inspired by nix-darwin evaluate-then-activate — **not** a full Nix `/nix/store` or flake protocol.

## Decisions

| Topic | Choice |
|-------|--------|
| Artifact | Generation under Application Support `state/generations/<id>/` |
| Packages | Snapshot desired taps/formulae/casks/mas (+ policy, macos) in `manifest.json` |
| Files | Content-addressed blobs in `files/<sha256>`; map in manifest |
| Links semantics | Declaration of managed paths; activation **copies** (regular files), never symlinks |
| Directories | Expand to per-file entries in the generation |
| Templates | Render at build time; store rendered bytes |
| Default UX | `apply` builds then activates; `build` only writes the generation |
| Rollback UI | History on disk only (keep last N); no TUI rollback in v0.3 |
| Prune | Keep last 5 generations after successful apply |

## Layout

```text
~/Library/Application Support/PourOver/state/
  generations/
    <id>/
      manifest.json
      files/
        <sha256>…   # blob bytes
  current            # text file: active generation id
  lock.json          # adds generation_id
```

`manifest.json` fields:

- `id`, `created_at`
- `packages`, `policy`, `macos` (from evaluated config)
- `files[]`: `{ target, mode, hash, kind, source }` where kind is `link|managed|template`

## Commands

| Command | Behavior |
|---------|----------|
| `pourover build` | Evaluate Lua; write new generation; print id; no live writes |
| `pourover plan` | Build generation (or reuse ephemeral); plan live vs generation |
| `pourover apply` | Build generation → activate → set `current` + lock `generation_id` |

## Activation

1. Package/PAM/defaults reconcile from generation’s desired packages/macos (same brew/`mas`/`defaults` path as today).
2. For each generation file entry: if live content hash ≠ blob hash (or missing/wrong type), write blob to target as a regular file (atomic temp+rename). Existing PourOver **symlinks** are replaced (backup per `policy.file_replace` when needed).
3. Owned-files / prune unchanged in spirit: targets from generation file map.

## Non-goals

- Full Nix store / build graph / substituters
- Keeping symlink mode as default
- Generation rollback TUI
