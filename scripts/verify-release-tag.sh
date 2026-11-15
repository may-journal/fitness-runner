#!/usr/bin/env bash
# Verifies the release tag matches the release version, then writes the version to the step output.
set -e
go run ./cmd/fitness-release verify-tag --root ..
printf 'version=%s\n' "${RELEASE_TAG#v}" >> "$GITHUB_OUTPUT"
