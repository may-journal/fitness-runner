#!/usr/bin/env bash
# Installs the checks the repo pins through githooks/checks-dir and points later steps at them with FITNESS_CHECKS_DIR.
set -e
checks="$(githooks/checks-dir)"
echo "FITNESS_CHECKS_DIR=$checks" >> "$GITHUB_ENV"
