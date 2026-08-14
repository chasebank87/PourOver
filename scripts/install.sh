#!/usr/bin/env bash
# PourOver installer — brew-style one-liner:
#   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/chasebank87/PourOver/main/scripts/install.sh)"
set -euo pipefail

REPO="chasebank87/PourOver"
BINARY="pourover"
BREW_INSTALL_URL="https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"

die() {
  echo "error: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

ohai() {
  echo "==> $*"
}

need_cmd curl
need_cmd tar
need_cmd uname
need_cmd mktemp

OS="$(uname -s)"
ARCH="$(uname -m)"
[[ "$OS" == "Darwin" ]] || die "PourOver currently supports macOS only (got $OS)"

case "$ARCH" in
  arm64|aarch64) GOARCH="arm64" ;;
  x86_64|amd64)  GOARCH="amd64" ;;
  *) die "unsupported architecture: $ARCH" ;;
esac

ensure_brew() {
  if command -v brew >/dev/null 2>&1; then
    ohai "Homebrew already installed: $(command -v brew)"
    return
  fi

  # Common post-install locations when brew isn't on PATH yet in this shell.
  for candidate in /opt/homebrew/bin/brew /usr/local/bin/brew; do
    if [[ -x "$candidate" ]]; then
      # shellcheck disable=SC1091
      eval "$("$candidate" shellenv)"
      ohai "Found Homebrew at $candidate"
      return
    fi
  done

  ohai "Homebrew not found; installing (NONINTERACTIVE=1)..."
  need_cmd bash
  NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL "${BREW_INSTALL_URL}")" || die "Homebrew install failed"

  for candidate in /opt/homebrew/bin/brew /usr/local/bin/brew; do
    if [[ -x "$candidate" ]]; then
      # shellcheck disable=SC1091
      eval "$("$candidate" shellenv)"
      break
    fi
  done

  command -v brew >/dev/null 2>&1 || die "Homebrew installed but brew is still not on PATH; open a new terminal and re-run"
  ohai "Homebrew installed: $(command -v brew)"
}

# Prefer Homebrew prefix bin when writable, else /usr/local/bin, else ~/.local/bin
install_dir() {
  local brew_prefix=""
  if command -v brew >/dev/null 2>&1; then
    brew_prefix="$(brew --prefix 2>/dev/null || true)"
  fi
  for d in "${brew_prefix:+$brew_prefix/bin}" /opt/homebrew/bin /usr/local/bin "$HOME/.local/bin"; do
    [[ -n "$d" ]] || continue
    if [[ -d "$d" && -w "$d" ]]; then
      echo "$d"
      return
    fi
    if mkdir -p "$d" 2>/dev/null && [[ -w "$d" ]]; then
      echo "$d"
      return
    fi
  done
  die "could not find a writable install directory"
}

ensure_brew

ohai "Fetching latest PourOver release..."
API="https://api.github.com/repos/${REPO}/releases/latest"
RELEASE_JSON="$(curl -fsSL "$API")" || die "failed to fetch latest release (create a GitHub Release first)"

TAG="$(printf '%s\n' "$RELEASE_JSON" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
[[ -n "$TAG" ]] || die "could not parse release tag"

ASSET_URL="$(printf '%s\n' "$RELEASE_JSON" | sed -n 's/.*"browser_download_url":[[:space:]]*"\([^"]*'"${GOARCH}"'[^"]*\.tar\.gz\)".*/\1/p' | head -1)"
[[ -n "$ASSET_URL" ]] || die "no darwin_${GOARCH} archive found in release ${TAG}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

ARCHIVE="$TMP/pourover.tar.gz"
ohai "Downloading ${ASSET_URL}..."
curl -fsSL -o "$ARCHIVE" "$ASSET_URL" || die "download failed"

tar -xzf "$ARCHIVE" -C "$TMP"
BIN_PATH="$(find "$TMP" -type f -name "$BINARY" | head -1)"
[[ -n "$BIN_PATH" && -x "$BIN_PATH" ]] || die "archive did not contain an executable named ${BINARY}"

DEST="$(install_dir)"
ohai "Installing ${BINARY} to ${DEST}/..."
cp "$BIN_PATH" "${DEST}/${BINARY}"
chmod +x "${DEST}/${BINARY}"

echo
echo "Installed PourOver ${TAG} -> ${DEST}/${BINARY}"
if ! command -v "$BINARY" >/dev/null 2>&1; then
  echo "Note: ${DEST} is not on your PATH. Add it, for example:"
  echo "  export PATH=\"${DEST}:\$PATH\""
  if command -v brew >/dev/null 2>&1; then
    echo "  eval \"\$(brew shellenv)\""
  fi
fi
echo
echo "Next steps:"
echo "  pourover init"
echo "  pourover doctor"
echo "  pourover plan"
