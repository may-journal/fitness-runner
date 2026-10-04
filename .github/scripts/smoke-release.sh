#!/usr/bin/env bash
# Run the packaged installer and real binaries with Go absent from PATH.
# cspell:ignore pipefail sha256sum shasum mktemp
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
tools=${FITNESS_TEST_TOOL_PATH:-$work/tools}
cache=${FITNESS_CACHE_DIR:-$work/cache}
mkdir -p "$tools" "$cache" "$work/repo"
for tool in bash awk mkdir mktemp rm rmdir sleep tar gzip sort uniq wc tr find mv shasum sha256sum uname git chmod; do
  path=$(command -v "$tool" || true)
  if [ -n "$path" ]; then ln -s "$path" "$tools/$tool"; fi
done
# All archives are genuine release artifacts; only transport is local before publishing.
for archive in "$root"/out/*.tar.gz; do
  digest=$(shasum -a 256 "$archive" | awk '{print $1}')
  name=${archive##*/fitness-}
  cp "$archive" "$cache/${name%.tar.gz}-$digest.tar.gz"
done
export PATH="$tools" FITNESS_CACHE_DIR="$cache"
if command -v go; then echo 'Go must not be available' >&2; exit 1; fi
cd "$work/repo"
git init -q
printf '# Example\n' > readme.md
git add readme.md
bash "$root/out/fitness.sh" -- --check=markdown-filename-kebab-case --all
bin=$(bash "$root/out/fitness.sh" --install-only)
"$bin/fitness" --help
printf '#!/bin/sh\nexec "%s" --check=markdown-filename-kebab-case --all\n' "$bin/fitness" > .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
git -c user.name=Fitness -c user.email=fitness@example.com commit -qm fixture
printf '# Example\n' > Bad_Name.md
git add Bad_Name.md
if bash "$root/out/fitness.sh" -- --check=markdown-filename-kebab-case --all; then
  echo 'invalid filename passed' >&2
  exit 1
fi

if git -c user.name=Fitness -c user.email=fitness@example.com commit -qm invalid; then
  echo 'hook allowed an invalid commit' >&2
  exit 1
fi
