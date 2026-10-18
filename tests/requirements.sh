#!/bin/sh
# Builds the candidate installer for this host's platform only, as the
# release build does for all four, and runs the requirement suite against
# it, 32 tests at once: they wait on subprocesses, not the CPU, and 32 keeps
# the process count well under the OS limit. No test leaves the machine:
# GitHub, npm, and tool answers are saved. With FITNESS_COVER set, the
# candidate is built with coverage and the run ends with coverage by package
# and in total.
set -eu
cd "$(dirname "$0")/.."
goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
platform="$goos-$goarch"
host_only="s/^\( *goos:\).*/\1 [$goos]/;s/^\( *goarch:\).*/\1 [$goarch]/"
mkdir -p out
for config in bundles tools; do
  # The filter rewrites one-line lists only, so stop on any other form.
  if ! grep -qE '^ *goos: \[' ".goreleaser-$config.yml" ||
    grep -E '^ *(goos|goarch):' ".goreleaser-$config.yml" | grep -qvE ': \[[a-z0-9, ]+\]$'; then
    echo "tests/requirements.sh: .goreleaser-$config.yml needs one-line goos and goarch lists" >&2
    exit 1
  fi
  sed "$host_only" ".goreleaser-$config.yml" >"out/host-$config.yml"
done
FITNESS_VERSION="$(go -C go run ./cmd/fitness-release version --root ..)"
export FITNESS_VERSION
cover=""
if [ -n "${FITNESS_COVER:-}" ]; then cover=-cover; fi
GOFLAGS="$cover" goreleaser release --clean --skip=publish --snapshot --config out/host-bundles.yml >out.log 2>&1
FITNESS_BUNDLE_HASHES="$(go -C go run ./cmd/fitness-release bundle-hashes --host --root ..)"
export FITNESS_BUNDLE_HASHES
rm -rf out/tools
GOFLAGS="$cover" goreleaser release --clean=false --skip=publish --snapshot --config out/host-tools.yml >>out.log 2>&1
go -C go run ./cmd/fitness-release assemble --host --root .. >>out.log 2>&1
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
