#!/usr/bin/env bash
# Prints each check's time, slowest first, from the performance record the
# suite kept. A suite that never wrote one already failed, so this stays quiet.
set -e
record="$RUNNER_TEMP/fitness-performance/fitness.json"
if [ -f "$record" ]; then go/bin/fitness perf "$record"; fi
