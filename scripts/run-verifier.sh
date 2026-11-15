#!/usr/bin/env bash
# Runs one command of the Go release verifier in VERIFIER, such as smoke or verify-download.
set -e
chmod +x "$VERIFIER"
"$VERIFIER" "$1"
