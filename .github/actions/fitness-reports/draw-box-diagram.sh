#!/usr/bin/env bash
# Draws the box diagram from the performance record, when fitness wrote one.
# Reads the fitness program's path from FITNESS.
set -eo pipefail
record="$RUNNER_TEMP/fitness-performance/fitness.json"
if [ -f "$record" ]; then
  "$FITNESS" boxes "$record" "$RUNNER_TEMP/fitness-performance/fitness.html"
fi
