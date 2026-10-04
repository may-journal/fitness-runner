#!/usr/bin/env bash
# Check the actual public URL from a fresh consumer checkout, without Go.
# cspell:ignore pipefail mktemp shasum sha256sum
set -euo pipefail
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir "$work/tools" "$work/repo"
for tool in bash curl awk mktemp rm chmod shasum sha256sum uname git; do
  path=$(command -v "$tool" || true)
  if [ -n "$path" ]; then ln -s "$path" "$work/tools/$tool"; fi
done
export PATH="$work/tools"
if command -v go; then echo 'Go must not be available' >&2; exit 1; fi
cd "$work/repo"
git init -q
printf '# Example\n' > readme.md
git add readme.md
bootstrap=$(curl -fsSL "https://github.com/may-journal/fitness-runner/releases/download/$GITHUB_REF_NAME/fitness.sh")
bash -c "$bootstrap" -- -- --check=markdown-filename-kebab-case --all
printf '# Example\n' > Bad_Name.md
git add Bad_Name.md
if bash -c "$bootstrap" -- -- --check=markdown-filename-kebab-case --all; then
  echo 'published binary lost a failing check status' >&2
  exit 1
fi
