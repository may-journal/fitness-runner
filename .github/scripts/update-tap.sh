#!/usr/bin/env bash
# Regenerates Formula/fitness.rb in may-journal/homebrew-tap from a published
# release's checksums and pushes it. Needs GH_TOKEN with push access to the
# tap. Usage: update-tap.sh go/v0.1.0
# cspell:ignore euo pipefail mktemp macos sha256 nocasematch
set -euo pipefail

tag="$1"
version="${tag#go/}"
version="${version#v}"
repo="may-journal/fitness-runner"
tap_repo="may-journal/homebrew-tap"

work="$(mktemp -d)"
gh release download "$tag" --repo "$repo" --pattern checksums.txt --dir "$work"

sha() {
  awk -v f="fitness-${version}-$1.tar.gz" '$2 == f || $2 == "*" f {print $1}' "$work/checksums.txt"
}
url() {
  echo "https://github.com/${repo}/releases/download/${tag}/fitness-${version}-$1.tar.gz"
}

cat > "$work/fitness.rb" <<FORMULA
class Fitness < Formula
  desc "Zero-dependency fitness checks - a runner plus 30 static check binaries"
  homepage "https://github.com/${repo}"
  version "${version}"
  license "MIT"

  on_macos do
    on_arm do
      url "$(url darwin-arm64)"
      sha256 "$(sha darwin-arm64)"
    end
    on_intel do
      url "$(url darwin-amd64)"
      sha256 "$(sha darwin-amd64)"
    end
  end

  on_linux do
    on_arm do
      url "$(url linux-arm64)"
      sha256 "$(sha linux-arm64)"
    end
    on_intel do
      url "$(url linux-amd64)"
      sha256 "$(sha linux-amd64)"
    end
  end

  def install
    bin.install Dir["fitness*"]
  end

  test do
    system "#{bin}/fitness-check-node-version", "--describe"
  end
end
FORMULA

grep -q 'sha256 "[a-f0-9]\{64\}"' "$work/fitness.rb" || {
  echo "checksum extraction failed" >&2
  exit 1
}

git clone "https://x-access-token:${GH_TOKEN}@github.com/${tap_repo}.git" "$work/tap"
mkdir -p "$work/tap/Formula"
cp "$work/fitness.rb" "$work/tap/Formula/fitness.rb"
cd "$work/tap"
git add Formula/fitness.rb
git -c user.name="fitness-release" -c user.email="release@may-journal" \
  commit -m "fitness ${version}" || echo "formula unchanged"
git push origin HEAD
