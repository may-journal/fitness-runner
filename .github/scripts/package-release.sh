#!/usr/bin/env bash
# Build the whole release, including a bootstrap bound to these exact hashes.
# cspell:ignore pipefail CGO GOOS GOARCH czf shasum getline
set -euo pipefail
cd "$(dirname "$0")/../.."
version=$(.github/scripts/release-tag.sh)
version=${version#go/v}
mkdir -p out
for os in darwin linux; do
  for arch in amd64 arm64; do
    dir="out/fitness-$version-$os-$arch"
    mkdir -p "$dir"
    (cd go && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -o "../$dir" ./cmd/...)
    tar -czf "$dir.tar.gz" -C out "${dir#out/}"
    rm -r "$dir"
  done
done
(cd out && shasum -a 256 ./*.tar.gz | sed 's|  ./|  |' > checksums.txt)
# The published script works offline against a warm cache without fetching a manifest.
awk -v version="$version" '
  /^version=/ { print "version=" version; next }
  /^bundle_hashes=/ {
    print "bundle_hashes=\047"
    while ((getline line < "out/checksums.txt") > 0) print line
    print "\047"
    next
  }
  { print }
' fitness.sh > out/fitness.sh
chmod +x out/fitness.sh
