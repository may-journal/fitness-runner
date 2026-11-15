#!/usr/bin/env bash
# Installs the pinned checks beside a source-built engine, and points later steps at them with FITNESS_CHECKS_DIR.
set -eo pipefail
checks="$("$(go env GOPATH)/bin/fitness-install" --version latest --install-only | sed -n 2p)"
if [ -n "$checks" ]; then echo "FITNESS_CHECKS_DIR=$checks" >> "$GITHUB_ENV"; fi
