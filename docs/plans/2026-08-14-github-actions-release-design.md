# GitHub Actions Release Publish — Design

**Date:** 2026-08-14  
**Status:** Approved

## Problem

PourOver install and `self-update` need GitHub Release assets (darwin amd64/arm64 tar.gz). There is already a GoReleaser workflow on `v*` tag push, plus an empty `workflow_dispatch` with no version input. We need a clear way to cut releases from either a local tag push or the Actions UI.

## Decision

Support **both**:

1. **Tag push** — `git tag vX.Y.Z && git push origin vX.Y.Z` triggers GoReleaser (unchanged responsibility).
2. **Workflow dispatch** — Actions → Release → Run workflow with a version input; the workflow **only creates and pushes an annotated tag on `main`**. The resulting tag push runs the publish job once.

## Why dispatch also runs GoReleaser

Tags pushed with `GITHUB_TOKEN` do **not** trigger new workflow runs. So `workflow_dispatch` creates the annotated tag on `main` **and** runs GoReleaser in the same workflow. Local/human tag pushes still use the `push: tags` path only (the `tag` job is skipped).

## Dispatch behavior

- Input: `version` (required), e.g. `0.1.0` or `v0.1.0`
- Normalize to `v` + semver-like `X.Y.Z` (optional prerelease suffix allowed if we keep validation simple: `v` + non-empty)
- Fail if tag already exists
- Checkout `main` with full history; create annotated tag at HEAD; push with `contents: write` and a token that can push tags (`GITHUB_TOKEN` is enough for same-repo tag push from Actions when permissions allow)

## Publish behavior (tag push)

- Existing GoReleaser job on `macos-latest`
- `permissions: contents: write`
- Build darwin amd64/arm64 with version/commit ldflags from `.goreleaser.yaml`

## Docs

Short README “Releasing” section covering both paths.

## Out of scope

- Homebrew formula tap automation
- Signed/notarized macOS binaries
- Auto-bump version from conventional commits
