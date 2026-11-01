#!/bin/sh
# Runs one shard of the requirement tests, so CI splits them across jobs.
#
# Usage: sh tests/requirements-shard.sh SHARD COUNT [go test flags...]
#
# Exit 2 means SHARD is out of range or holds no tests; tests/requirements.sh
# sets FITNESS_INSTALL and FITNESS_CACHE_DIR.
set -eu
cd "$(dirname "$0")/.."
shard="$1"
count="$2"
shift 2
case "$shard$count" in *[!0-9]*) shard=0 ;; esac
if [ "$shard" -lt 1 ] || [ "$shard" -gt "$count" ]; then
  echo "requirements-shard: SHARD must be from 1 to COUNT, got $shard of $count" >&2
  exit 2
fi
names="$(go -C go test ./requirements -list '^Test' |
  grep '^Test' | sort |
  awk -v shard="$shard" -v count="$count" '(NR - shard) % count == 0' |
  paste -sd '|' -)"
if [ -z "$names" ]; then
  echo "requirements-shard: shard $shard of $count holds no tests" >&2
  exit 2
fi
exec go -C go test ./requirements -count=1 -parallel 32 -run "^($names)\$" "$@"
