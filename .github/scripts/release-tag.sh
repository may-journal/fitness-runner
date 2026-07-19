#!/usr/bin/env bash
# Prints the release tag derived from the newest CHANGELOG.md heading.
# The heading IS the version; this maps it to Go-compatible semver:
#   ### 2026.07.19.0837  ->  go/v0.20260719.837
# (major pinned at 0 because Go reserves majors >= 2 for /vN module paths;
# minor is the date, patch the minute, leading zeros stripped, order kept)
# cspell:ignore euo pipefail gsub substr
set -euo pipefail

awk 'sub(/^### /, "") {
  gsub(/\./, "", $1)
  print "go/v0." substr($1, 1, 8) "." int(substr($1, 9))
  exit
}' "$(dirname "$0")/../../CHANGELOG.md"
