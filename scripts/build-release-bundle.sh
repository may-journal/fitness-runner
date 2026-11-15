#!/usr/bin/env bash
# Builds the release bundles and tools with GoReleaser and assembles the release, as a snapshot when SNAPSHOT is true.
set -e
FITNESS_VERSION="$(go -C go run ./cmd/fitness-release version --root ..)"
export FITNESS_VERSION
args=()
if [ "$SNAPSHOT" = true ]; then args+=(--snapshot); fi
goreleaser release --clean --skip=publish "${args[@]}" --config .goreleaser-bundles.yml
FITNESS_BUNDLE_HASHES="$(go -C go run ./cmd/fitness-release bundle-hashes --root ..)"
export FITNESS_BUNDLE_HASHES
goreleaser release --clean --skip=publish "${args[@]}" --config .goreleaser-tools.yml
go -C go run ./cmd/fitness-release assemble --root ..
