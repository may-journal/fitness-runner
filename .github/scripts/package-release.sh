#!/usr/bin/env bash
# Build the whole release, including a bootstrap bound to these exact hashes.
# cspell:ignore pipefail CGO GOOS GOARCH czf shasum getline
set -euo pipefail
cd "$(dirname "$0")/../.."
version=$(.github/scripts/release-tag.sh)
version=${version#go/v}
rm -rf out
mkdir -p out
for os in darwin linux; do
  for arch in amd64 arm64; do
    dir="out/fitness-$version-$os-$arch"
    mkdir -p "$dir"
    (cd go && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -o "../$dir" ./cmd/...)
    rm "$dir/fitness-install"
    tar -czf "$dir.tar.gz" -C out "${dir#out/}"
    rm -r "$dir"
  done
done
(cd out && shasum -a 256 ./*.tar.gz | sed 's|  ./|  |' > checksums.txt)
# The typed installer carries the hashes for every bundle in this release.
bundle_hashes=$(awk -v prefix="fitness-$version-" '{
  name=$2; sub("^" prefix, "", name); sub(/\.tar\.gz$/, "", name)
  printf "%s%s=%s", separator, name, $1; separator=","
}' out/checksums.txt)
for os in darwin linux; do
  for arch in amd64 arm64; do
    (cd go && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
      -ldflags "-X main.version=$version -X main.bundleHashes=$bundle_hashes" \
      -o "../out/fitness-install-$version-$os-$arch" ./cmd/fitness-install)
  done
done
(cd out && shasum -a 256 ./*.tar.gz ./fitness-install-* | sed 's|  ./|  |' > checksums.txt)
# The shell entry point only verifies and starts the compiled installer.
awk -v version="$version" '
  /^version=/ { print "version=" version; next }
  /^installer_hashes=/ {
    print "installer_hashes=\047"
    while ((getline line < "out/checksums.txt") > 0) print line
    print "\047"
    next
  }
  { print }
' fitness.sh > out/fitness.sh
chmod +x out/fitness.sh
