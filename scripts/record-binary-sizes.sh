#!/usr/bin/env bash
# Records binary sizes and install time for PLATFORM into the performance folder, using the VERIFIER program.
set -e
mkdir -p "$RUNNER_TEMP/fitness-performance"
"$VERIFIER" record > "$RUNNER_TEMP/release-$PLATFORM.json"
mv "$RUNNER_TEMP/release-$PLATFORM.json" "$RUNNER_TEMP/fitness-performance/"
