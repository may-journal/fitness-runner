#!/usr/bin/env bash
# Installs the pinned SwiftLint, checking the zip against its published SHA-256 and caching it under a versioned name.
set -e
zip="$HOME/.cache/fitness-swiftlint/${ASSET}-${VERSION}.zip"
want=
for attempt in 1 2 3; do
  if want=$(gh api "repos/realm/SwiftLint/releases/tags/${VERSION}" \
    --jq ".assets[] | select(.name == \"${ASSET}.zip\") | .digest"); then
    break
  fi
  sleep "$attempt"
done
want="${want#sha256:}"
if [ -z "$want" ]; then
  echo "::error::SwiftLint ${VERSION} publishes no SHA-256 for ${ASSET}.zip"
  exit 1
fi
if [ ! -f "$zip" ] || ! echo "$want  $zip" | shasum -a 256 -c - >/dev/null; then
  mkdir -p "$(dirname "$zip")"
  part="$zip.$GITHUB_RUN_ID.$GITHUB_JOB.$$"
  curl -fsSL --retry 3 -o "$part" \
    "https://github.com/realm/SwiftLint/releases/download/${VERSION}/${ASSET}.zip"
  if ! echo "$want  $part" | shasum -a 256 -c -; then
    rm -f "$part"
    echo "::error::SwiftLint ${VERSION} ${ASSET}.zip does not match its published SHA-256"
    exit 1
  fi
  mv -f "$part" "$zip"
fi
rm -rf "$RUNNER_TEMP/swiftlint"
unzip -q -o "$zip" -d "$RUNNER_TEMP/swiftlint"
chmod +x "$RUNNER_TEMP/swiftlint/swiftlint"
echo "$RUNNER_TEMP/swiftlint" >> "$GITHUB_PATH"
"$RUNNER_TEMP/swiftlint/swiftlint" version
