#!/usr/bin/env bash
# Fetch a release bundle; compilation belongs to the release pipeline.
# cspell:ignore pipefail mktemp shasum sha256sum xzf tzf tvzf uname aarch64 esac
set -euo pipefail

version='0.20261004.900'
# Release publishing fills this value with the bundle hashes.
bundle_hashes=''
release_root='https://github.com/may-journal/fitness-runner/releases'
mode=run

fail() { printf 'fitness: %s\n' "$*" >&2; exit 1; }
usage() {
  echo 'fitness.sh [--version VERSION|latest] [--install-only] [-- RUNNER_ARGS...]'
}
while [ "$#" -gt 0 ]; do
  case "$1" in
    --version) [ "$#" -ge 2 ] || fail '--version needs a value'; version="$2"; bundle_hashes=''; shift 2 ;;
    --install-only) mode=install; shift ;;
    --help) usage; exit 0 ;;
    --) shift; break ;;
    *) fail "unknown installer option: $1 (put runner arguments after --)" ;;
  esac
done

if [ "$version" = latest ]; then
  latest=$(curl -fsSL --retry 3 "$release_root/latest" -o /dev/null -w '%{url_effective}')
  version=${latest##*/}
  version=${version#go%2F}
  printf 'fitness: latest resolved to %s\n' "$version" >&2
fi
version=${version#go/v}
version=${version#v}
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "invalid release version: $version"
case "$(uname -s)/$(uname -m)" in
  Linux/x86_64) platform=linux-amd64 ;;
  Linux/aarch64|Linux/arm64) platform=linux-arm64 ;;
  Darwin/x86_64) platform=darwin-amd64 ;;
  Darwin/arm64) platform=darwin-arm64 ;;
  *) fail 'supported platforms: Linux/macOS on amd64/arm64' ;;
esac
if command -v sha256sum >/dev/null 2>&1; then
  hash_tool=(sha256sum)
else
  command -v shasum >/dev/null 2>&1 || fail 'install sha256sum or shasum'
  hash_tool=(shasum -a 256)
fi
hash_file() { "${hash_tool[@]}" "$1" | awk '{print $1}'; }

cache=${FITNESS_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/fitness}
umask 077
mkdir -p "$cache"
cache=$(cd "$cache" && pwd -P)
work=$(mktemp -d "$cache/.install.XXXXXX")
lock=''
cleanup() {
  rm -rf "$work"
  if [ -n "$lock" ]; then rmdir "$lock"; fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
asset="fitness-$version-$platform.tar.gz"
if [ -z "$bundle_hashes" ]; then
  bundle_hashes=$(curl -fsSL --retry 3 "$release_root/download/go/v$version/checksums.txt")
fi
printf '%s\n' "$bundle_hashes" > "$work/checksums"
digest=$(awk -v asset="$asset" '$2 == asset {print $1}' "$work/checksums")
[[ "$digest" =~ ^[a-f0-9]{64}$ ]] || fail "missing or invalid checksum for $asset"
key="$version-$platform-$digest"
archive="$cache/$key.tar.gz"
install="$cache/$key"
for ((attempt=0; ; attempt++)); do
  if mkdir "$cache/$key.lock" 2>/dev/null; then lock="$cache/$key.lock"; break; fi
  [ "$attempt" -lt 60 ] || fail "cache is busy: $cache/$key.lock"
  sleep 1
done
if [ ! -f "$archive" ] || [ "$(hash_file "$archive")" != "$digest" ]; then
  curl -fsSL --retry 3 "$release_root/download/go/v$version/$asset" -o "$work/bundle.tar.gz"
  [ "$(hash_file "$work/bundle.tar.gz")" = "$digest" ] || fail "checksum mismatch for $asset"
  mv "$work/bundle.tar.gz" "$archive"
fi

# Only flat regular binary files and their manifest may enter the cache.
top="fitness-$version-$platform"
tar -tzf "$archive" > "$work/entries"
tar -tvzf "$archive" > "$work/types"
while IFS= read -r entry; do
  case "$entry" in
    "$top/"|"$top/fitness"|"$top/files.sha256") ;;
    "$top/fitness-"*) [[ "${entry#"$top/"}" =~ ^fitness-[a-z0-9-]+$ ]] || fail "unsafe archive path: $entry" ;;
    *) fail "unsafe archive path: $entry" ;;
  esac
done < "$work/entries"
while read -r mode_bits rest; do
  case "$mode_bits" in -*) ;; d*) ;; *) fail 'archive contains a link or special file' ;; esac
done < "$work/types"
[ "$(sort "$work/entries" | uniq -d | wc -l | tr -d ' ')" = 0 ] || fail 'archive has duplicate paths'
mkdir "$work/unpack"
tar -xzf "$archive" -C "$work/unpack"
staged="$work/unpack/$top"
[ -x "$staged/fitness" ] || fail 'archive lacks an executable runner'
# Derive trusted file hashes from the verified archive, never from the cache.
: > "$work/files"
for binary in "$staged"/fitness*; do
  [ -f "$binary" ] && [ -x "$binary" ] || fail 'archive contains a non-executable binary'
  printf '%s %s\n' "$(hash_file "$binary")" "${binary##*/}" >> "$work/files"
done
cache_valid() {
  [ -d "$install" ] && [ ! -L "$install" ] || return 1
  while read -r expected name; do
    [ -f "$install/$name" ] && [ ! -L "$install/$name" ] && [ -x "$install/$name" ] || return 1
    [ "$(hash_file "$install/$name")" = "$expected" ] || return 1
  done < "$work/files"
  [ "$(find "$install" -mindepth 1 -maxdepth 1 | wc -l)" -eq "$(find "$staged" -mindepth 1 -maxdepth 1 | wc -l)" ]
}
if ! cache_valid; then
  rm -rf "$install"
  mv "$staged" "$install"
fi
rmdir "$lock"
lock=''
printf 'fitness: version=%s platform=%s sha256=%s\n' "$version" "$platform" "$digest" >&2
if [ "$mode" = install ]; then
  printf '%s\n' "$install"
else
  "$install/fitness" "$@"
fi
