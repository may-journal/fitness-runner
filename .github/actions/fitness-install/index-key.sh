#!/usr/bin/env bash
# Names the checks cache keys from the SHA-256 of the release index, which names every check version a release holds.
# The keys are base for any set, index for this index's sets and job for this job's; an unread index names none.
set -eo pipefail
url=https://github.com/may-journal/fitness-checks/releases/download/fitness-check-index/index.json
if ! hash="$(curl -fsSL --retry 3 "$url" | shasum -a 256)"; then
  printf '::warning::Could not read the release index, so the checks cache is not restored.\n'
  exit 0
fi
base="fitness-checks-$RUNNER_OS-$RUNNER_ARCH-"
index="$base${hash:0:64}-"
printf 'base=%s\nindex=%s\njob=%s\n' "$base" "$index" "$index$GITHUB_JOB-" >> "$GITHUB_OUTPUT"
