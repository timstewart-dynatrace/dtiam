#!/usr/bin/env sh
# install.sh — install the dtiam CLI on macOS or Linux.
#
# dtiam is an independent, community-developed tool and is NOT produced,
# endorsed, or supported by Dynatrace.
#
#   curl -fsSL https://raw.githubusercontent.com/jtimothystewart/dtiam/main/install.sh | sh
#
# Environment variables:
#   DTIAM_VERSION    version to install (default: latest release)
#   DTIAM_INSTALL_DIR  install directory (default: /usr/local/bin, or
#                      $HOME/.local/bin when the former is not writable)

set -eu

REPO="jtimothystewart/dtiam"
BINARY="dtiam"

err() { printf '%s\n' "error: $*" >&2; exit 1; }
info() { printf '%s\n' "$*" >&2; }

need() { command -v "$1" >/dev/null 2>&1 || err "$1 is required but not installed"; }

need uname
need tar

# curl or wget, whichever is present.
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_to() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_to() { wget -qO "$2" "$1"; }
else
  err "curl or wget is required"
fi

# --- platform detection -------------------------------------------------------

os=$(uname -s)
case "$os" in
  Darwin) os=darwin ;;
  Linux)  os=linux ;;
  *) err "unsupported operating system: $os (supported: macOS, Linux)" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) err "unsupported architecture: $arch (supported: amd64, arm64)" ;;
esac

# --- version resolution -------------------------------------------------------

version="${DTIAM_VERSION:-}"
if [ -z "$version" ]; then
  info "Resolving latest release..."
  version=$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | tr ',' '\n' | grep '"tag_name"' | head -n1 | cut -d'"' -f4) || true
  [ -n "$version" ] || err "could not determine the latest version; set DTIAM_VERSION"
fi
# Accept both "1.2.3" and "v1.2.3".
case "$version" in v*) tag="$version"; bare="${version#v}" ;; *) tag="v$version"; bare="$version" ;; esac

# --- install directory --------------------------------------------------------

install_dir="${DTIAM_INSTALL_DIR:-}"
if [ -z "$install_dir" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then
    install_dir=/usr/local/bin
  else
    install_dir="$HOME/.local/bin"
    info "/usr/local/bin is not writable; installing to $install_dir"
  fi
fi
mkdir -p "$install_dir" || err "cannot create $install_dir"

# --- download and verify ------------------------------------------------------

archive="${BINARY}_${bare}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$tag"

tmp=$(mktemp -d) || err "cannot create a temporary directory"
trap 'rm -rf "$tmp"' EXIT INT TERM

info "Downloading $archive ($tag)..."
fetch_to "$base/$archive" "$tmp/$archive" \
  || err "download failed: $base/$archive"

# Checksum verification, when a sha256 tool and the checksums file are available.
if fetch_to "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then
    sha_cmd="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    sha_cmd="shasum -a 256"
  else
    sha_cmd=""
  fi

  if [ -n "$sha_cmd" ]; then
    expected=$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')
    if [ -n "$expected" ]; then
      actual=$($sha_cmd "$tmp/$archive" | awk '{print $1}')
      [ "$expected" = "$actual" ] || err "checksum mismatch for $archive
  expected: $expected
  actual:   $actual"
      info "Checksum verified."
    else
      info "warning: $archive not listed in checksums.txt; skipping verification"
    fi
  else
    info "warning: no sha256 tool found; skipping checksum verification"
  fi
else
  info "warning: checksums.txt unavailable; skipping checksum verification"
fi

# --- install ------------------------------------------------------------------

tar -xzf "$tmp/$archive" -C "$tmp" || err "failed to extract $archive"
[ -f "$tmp/$BINARY" ] || err "$BINARY not found in the archive"

if [ -w "$install_dir" ]; then
  install -m 0755 "$tmp/$BINARY" "$install_dir/$BINARY" 2>/dev/null \
    || { cp "$tmp/$BINARY" "$install_dir/$BINARY" && chmod 0755 "$install_dir/$BINARY"; }
else
  info "Elevated permissions required to write to $install_dir"
  sudo install -m 0755 "$tmp/$BINARY" "$install_dir/$BINARY" 2>/dev/null \
    || { sudo cp "$tmp/$BINARY" "$install_dir/$BINARY" && sudo chmod 0755 "$install_dir/$BINARY"; }
fi

info "Installed $BINARY $tag to $install_dir/$BINARY"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) info ""
     info "$install_dir is not on your PATH. Add it:"
     info "  export PATH=\"$install_dir:\$PATH\"" ;;
esac

info ""
info "Next steps:"
info "  $BINARY --help"
info "  $BINARY doctor     # verify configuration and connectivity"
