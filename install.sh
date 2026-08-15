#!/bin/sh
set -eu

REPO="kuyacarlo/ssh-profile"
BIN="git-ssh"
DEST="${DEST:-/usr/local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

version="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -m1 '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
[ -n "$version" ] || { echo "could not resolve latest release" >&2; exit 1; }

url="https://github.com/${REPO}/releases/download/${version}/${BIN}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Installing ${BIN} ${version} (${os}/${arch}) to ${DEST}..."
curl -fsSL "$url" | tar -xz -C "$tmp"

if [ -w "$DEST" ]; then
  mv "$tmp/${BIN}" "$DEST/${BIN}"
else
  sudo mv "$tmp/${BIN}" "$DEST/${BIN}"
fi

echo "Installed: $("$DEST/$BIN" version 2>/dev/null || echo "${DEST}/${BIN}")"
