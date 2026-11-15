#!/usr/bin/env bash
# Moves the major version tag, such as v1, to RELEASE_TAG when that release is the newest.
set -e
latest="$(gh release view --json tagName --jq .tagName)"
if [ "$latest" = "$RELEASE_TAG" ]; then
  major="${RELEASE_TAG%%.*}"
  git tag -f "$major" "$(git rev-list -n 1 "$RELEASE_TAG")"
  git push -f origin "refs/tags/$major"
fi
