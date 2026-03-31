#!/usr/bin/env bash
set -euo pipefail

REPO="vmaliwal/airlock"
BIN_NAME="airlock"
INSTALL_DIR="${AIRLOCK_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${AIRLOCK_VERSION:-latest}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

mkdir -p "$INSTALL_DIR"

latest_tag() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tag_name",""))'
}

version="$VERSION"
if [[ "$version" == "latest" ]]; then
  version="$(latest_tag || true)"
fi

if [[ -n "$version" ]]; then
  tag="${version#v}"
  asset="airlock_${tag}_${os}_${arch}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${version}/${asset}"
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT
  if curl -fsSL "$url" -o "$tmpdir/$asset"; then
    tar -xzf "$tmpdir/$asset" -C "$tmpdir"
    install -m 0755 "$tmpdir/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
    echo "installed $BIN_NAME to $INSTALL_DIR/$BIN_NAME"
    exit 0
  fi
fi

if command -v go >/dev/null 2>&1; then
  echo "release binary unavailable; falling back to go install" >&2
  # Inject the version tag at link time so 'airlock metrics' shows the real version.
  ldflags=""
  if [[ -n "${version:-}" ]]; then
    ldflags="-ldflags=-X github.com/vmaliwal/airlock/internal/research.BuildVersion=${version}"
  fi
  GOBIN="$INSTALL_DIR" go install ${ldflags:+"$ldflags"} github.com/vmaliwal/airlock/cmd/airlock@latest
  echo "installed $BIN_NAME to $INSTALL_DIR/$BIN_NAME"
  exit 0
fi

echo "no release binary available and Go toolchain not found" >&2
echo "install Go 1.23+ and run: go install github.com/vmaliwal/airlock/cmd/airlock@latest" >&2
exit 1
