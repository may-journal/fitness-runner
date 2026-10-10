#!/bin/sh
# Builds the candidate installer, as the release build does, and runs the
# requirement suite against it, 32 tests at once: they wait on subprocesses,
# not the CPU, and 32 keeps the process count well under the OS limit. No
# test leaves the machine: GitHub, npm, and tool answers are saved. With
# FITNESS_COVER set, the candidate is built with coverage and the run ends
# with coverage by package and in total.
set -eu
cd "$(dirname "$0")/.."
platform="$(go env GOOS)-$(go env GOARCH)"
FITNESS_VERSION="$(go -C go run ./cmd/fitness-release version --root ..)"
export FITNESS_VERSION
cover=""
if [ -n "${FITNESS_COVER:-}" ]; then cover=-cover; fi
GOFLAGS="$cover" goreleaser release --clean --skip=publish --snapshot --config .goreleaser-bundles.yml >out.log 2>&1
FITNESS_BUNDLE_HASHES="$(go -C go run ./cmd/fitness-release bundle-hashes --root ..)"
export FITNESS_BUNDLE_HASHES
rm -rf out/tools
GOFLAGS="$cover" goreleaser release --clean=false --skip=publish --snapshot --config .goreleaser-tools.yml >>out.log 2>&1
go -C go run ./cmd/fitness-release assemble --root .. >>out.log 2>&1
rm out.log
FITNESS_CACHE_DIR="$PWD/out/requirements-cache"
export FITNESS_CACHE_DIR
"out/fitness-release-$platform" smoke >/dev/null
FITNESS_INSTALL="$(ls "$PWD"/out/fitness-install-*-"$platform")"
export FITNESS_INSTALL
if [ -z "$cover" ]; then
  exec go -C go test ./requirements -count=1 -parallel 32 "$@"
fi
rm -rf out/coverage && mkdir -p out/coverage
GOCOVERDIR="$PWD/out/coverage" go -C go test ./requirements -count=1 -parallel 32 "$@"
go -C go tool covdata percent -i ../out/coverage
go -C go tool covdata textfmt -i ../out/coverage -o ../out/coverage.txt
go -C go tool cover -func ../out/coverage.txt | tail -1
