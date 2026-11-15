#!/usr/bin/env bash
# Usage: install-source.sh <ref>, which builds fitness from the ref and puts it on PATH.
# A merged pull request's head is gone with its branch, so a failed build retries on the merge commit.
set -euo pipefail

ref="${1:?usage: install-source.sh <ref>}"
module=github.com/may-journal/fitness-runner/go/cmd/...

if ! go install "$module@$ref"; then
  number="$(jq -r '.pull_request.number // empty' "$GITHUB_EVENT_PATH")"
  merge=""
  if [ -n "$number" ]; then
    merge="$(gh api "repos/$GITHUB_REPOSITORY/pulls/$number" --jq 'select(.merged) | .merge_commit_sha')"
  fi
  if [ -z "$merge" ]; then
    echo "::error::go install of $ref failed, and it is not the head of a merged pull request, so no other commit stands in for it."
    exit 1
  fi
  echo "::notice::Pull request $number merged and its head $ref is gone, so fitness builds from its merge commit $merge."
  go install "$module@$merge"
fi
echo "$(go env GOPATH)/bin" >> "$GITHUB_PATH"
