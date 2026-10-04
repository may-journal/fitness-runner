#!/usr/bin/env bash
# Only fetch and start the compiled installer; release/cache/archive logic is Go.
# cspell:ignore pipefail mktemp shasum sha256sum uname aarch64 esac
set -euo pipefail
version='0.20261004.913'
installer_hashes=''
base="https://github.com/may-journal/fitness-runner/releases/download/go/v$version"
case "$(uname -s)/$(uname -m)" in
  Linux/x86_64) platform=linux-amd64 ;;
  Linux/aarch64|Linux/arm64) platform=linux-arm64 ;;
  Darwin/x86_64) platform=darwin-amd64 ;;
  Darwin/arm64) platform=darwin-arm64 ;;
  *) echo 'fitness: use Linux or macOS on amd64 or arm64' >&2; exit 1 ;;
esac
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
asset="fitness-install-$version-$platform"
if [ -z "$installer_hashes" ]; then
  installer_hashes=$(curl -fsSL --retry 3 "$base/checksums.txt")
fi
expected=$(printf '%s\n' "$installer_hashes" | awk -v asset="$asset" '$2 == asset {print $1}')
[[ "$expected" =~ ^[a-f0-9]{64}$ ]] || { echo 'fitness: missing installer checksum' >&2; exit 1; }
curl -fsSL --retry 3 "$base/$asset" -o "$work/fitness-install"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$work/fitness-install" | awk '{print $1}')
else
  actual=$(shasum -a 256 "$work/fitness-install" | awk '{print $1}')
fi
[ "$actual" = "$expected" ] || { echo 'fitness: installer checksum mismatch' >&2; exit 1; }
chmod +x "$work/fitness-install"
"$work/fitness-install" "$@"
