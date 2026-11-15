#!/usr/bin/env bash
# Runs the full suite, with the pinned checks, through the candidate Go
# installer for PLATFORM.
set -e
installer=(out/fitness-install-*-"$PLATFORM")
chmod +x "${installer[0]}"
"${installer[0]}" --with-checks -- --policy=external
