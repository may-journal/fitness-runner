#!/bin/sh
# Runs one shard of the requirement tests, so CI splits them across jobs.
#
# Usage: sh tests/requirements-shard.sh SHARD COUNT [go test flags...]
#
# Inputs: SHARD, from 1 to COUNT, picks every COUNT-th test of the sorted
# list, starting at test SHARD. FITNESS_INSTALL names the candidate
# fitness-install, and FITNESS_CACHE_DIR a cache its verifier's smoke run
# seeded, as tests/requirements.sh sets them. Output: go test's own output
# for the shard's tests, 32 at once, and its exit code.
set -eu
cd "$(dirname "$0")/.."
shard="$1"
count="$2"
shift 2
names="$(go -C go test ./requirements -list '^Test' |
  grep '^Test' | sort |
  awk -v shard="$shard" -v count="$count" '(NR - shard) % count == 0' |
  paste -sd '|' -)"
exec go -C go test ./requirements -count=1 -parallel 32 -run "^($names)\$" "$@"
