#!/usr/bin/env bash
# Writes the shallowest go.mod, or nothing, to the step output as file, so setup-go can use the runner's Go.
set -eo pipefail
file=$(git ls-files -- go.mod '*/go.mod' | awk -F/ '{ print NF, $0 }' | sort -n | head -n 1 | cut -d' ' -f2-)
echo "file=${file:-}" >> "$GITHUB_OUTPUT"
