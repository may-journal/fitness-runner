#!/usr/bin/env bash
# Writes the release notes from the changelog entry files onto the draft release, so a failure leaves it unpublished.
set -e
chmod +x out/fitness-release-linux-amd64
out/fitness-release-linux-amd64 release-notes
