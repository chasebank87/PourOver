# GitHub Actions Release Publish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let maintainers publish PourOver releases via `v*` tag push or Actions workflow_dispatch that tags `main`.

**Architecture:** Split responsibilities: `workflow_dispatch` normalizes a version, creates an annotated tag on `main`, and pushes it; the existing `push: tags: v*` job runs GoReleaser once. README documents both paths.

**Tech Stack:** GitHub Actions, GoReleaser v2, existing `.goreleaser.yaml`

---

### Task 1: Update release workflow — tag job + publish job

**Files:**
- Modify: `.github/workflows/release.yml`

**Step 1: Replace workflow with two clear jobs**

Use this content for `.github/workflows/release.yml`:

```yaml
name: release

on:
  push:
    tags:
      - "v*"
  workflow_dispatch:
    inputs:
      version:
        description: "Release version (e.g. 0.1.0 or v0.1.0)"
        required: true
        type: string

permissions:
  contents: write

jobs:
  tag:
    if: github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    steps:
      - name: Normalize version
        id: ver
        run: |
          raw="${{ inputs.version }}"
          raw="${raw#v}"
          if [[ ! "$raw" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
            echo "Invalid version: ${{ inputs.version }} (want X.Y.Z or vX.Y.Z)"
            exit 1
          fi
          echo "tag=v${raw}" >> "$GITHUB_OUTPUT"

      - uses: actions/checkout@v4
        with:
          ref: main
          fetch-depth: 0

      - name: Create and push tag
        env:
          TAG: ${{ steps.ver.outputs.tag }}
        run: |
          if git rev-parse "$TAG" >/dev/null 2>&1; then
            echo "Tag $TAG already exists"
            exit 1
          fi
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git tag -a "$TAG" -m "Release $TAG"
          git push origin "$TAG"

  goreleaser:
    if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Step 2: Sanity-check YAML locally**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`  
(or skip if PyYAML missing — visual review is fine)

**Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: split release into tag dispatch and GoReleaser on tag push"
```

---

### Task 2: Document releasing in README

**Files:**
- Modify: `README.md` (after Install or near CI mention)

**Step 1: Add a short Releasing section**

```markdown
## Releasing

Publish a GitHub Release (darwin archives for install / `self-update`) via either:

1. **Tag push**
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
2. **Actions UI** — Actions → **release** → Run workflow → enter `0.1.0` or `v0.1.0`.  
   This tags current `main`; the tag push runs GoReleaser.
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: document tag push and Actions release flows"
```

---

### Task 3: Verify (local + optional remote)

**Step 1:** Confirm `.goreleaser.yaml` still stamps `version.Version` / `version.Commit`.

**Step 2:** After merge/push to `main`, optionally dry-run from a fork or wait for first real `v*` tag. Do not create a real release tag unless the user asks.

**Step 3:** Final commit only if any leftover fixes; otherwise done.
