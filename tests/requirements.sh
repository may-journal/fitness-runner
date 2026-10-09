#!/bin/sh
# Builds the candidate installer, as the release build does, and runs the
# requirement suite against it, one test per core (go test's default). With FITNESS_QUICK
# set, tests that reach GitHub, the npm registry, or a real external tool
# skip; every CI smoke job still runs them.
set -eu
cd "$(dirname "$0")/.."
platform="$(go env GOOS)-$(go env GOARCH)"
FITNESS_VERSION="$(go -C go run ./cmd/fitness-release version --root ..)"
export FITNESS_VERSION
goreleaser release --clean --skip=publish --snapshot --config .goreleaser-bundles.yml >out.log 2>&1
FITNESS_BUNDLE_HASHES="$(go -C go run ./cmd/fitness-release bundle-hashes --root ..)"
export FITNESS_BUNDLE_HASHES
rm -rf out/tools
goreleaser release --clean=false --skip=publish --snapshot --config .goreleaser-tools.yml >>out.log 2>&1
go -C go run ./cmd/fitness-release assemble --root .. >>out.log 2>&1
rm out.log
FITNESS_CACHE_DIR="$PWD/out/requirements-cache"
export FITNESS_CACHE_DIR
"out/fitness-release-$platform" smoke >/dev/null
if [ -z "${FITNESS_QUICK:-}" ]; then
  sh tests/tools/install.sh >/dev/null
fi
FITNESS_INSTALL="$(ls "$PWD"/out/fitness-install-*-"$platform")"
export FITNESS_INSTALL
go -C go test ./requirements -count=1 "$@"
