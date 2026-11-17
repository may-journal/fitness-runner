#!/usr/bin/env bash
# Names the key the checks set saves under: the job's prefix and a hash of the programs it holds.
# A set restored from an older index first drops each program this run did not link, so old versions leave the cache.
set -eo pipefail
cache="${XDG_CACHE_HOME:-$HOME/.cache}/fitness"
[ -d "$cache/checks" ] || exit 0
if [[ -z "$FITNESS_RESTORED_KEY" || "$FITNESS_RESTORED_KEY" != "$FITNESS_INDEX_KEY"* ]]; then
  used="$(find "$cache/check-sets" -type l -exec readlink {} + 2>/dev/null | while read -r program; do dirname "$program"; done | sort -u || true)"
  for dir in "$cache"/checks/*/; do
    grep -qxF "${dir%/}" <<< "$used" || rm -rf "$dir"
  done
fi
programs="$(cd "$cache/checks" && find . -type f | sort)"
[ -n "$programs" ] || exit 0
hash="$(shasum -a 256 <<< "$programs")"
printf 'key=%s%s\n' "$FITNESS_JOB_KEY" "${hash:0:16}" >> "$GITHUB_OUTPUT"
