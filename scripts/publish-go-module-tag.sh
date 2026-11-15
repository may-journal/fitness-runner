#!/usr/bin/env bash
# Tags the release commit as the Go module version, and fails when an existing tag names another commit.
set -e
release_sha="$(git rev-list -n 1 "$RELEASE_TAG")"
module_tag="go/$RELEASE_TAG"
if git rev-parse --verify "$module_tag" >/dev/null 2>&1; then
  test "$(git rev-list -n 1 "$module_tag")" = "$release_sha"
else
  git tag "$module_tag" "$release_sha"
  git push origin "refs/tags/$module_tag"
fi
