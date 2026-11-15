#!/usr/bin/env bash
# Usage: install-source.sh <ref>, which builds fitness from the ref and puts it on PATH.
# A merged pull request's head is gone with its branch, so a failed build retries on the merge commit.
set -euo pipefail

ref="${1:?usage: install-source.sh <ref>}"
module=github.com/may-journal/fitness-runner/go/cmd/...

if ! go install "$module@$ref"; then
  number="$(jq -r '.pull_request.number // .issue.number // empty' "$GITHUB_EVENT_PATH")"
  merge=""
  if [ -n "$number" ] && [ "$GITHUB_REPOSITORY" = may-journal/fitness-runner ]; then
    # A failed lookup leaves merge empty, so the error below still names the failed build.
    merge="$(gh api "repos/$GITHUB_REPOSITORY/pulls/$number" | jq -r --arg ref "$ref" 'select(.merged and .head.sha == $ref) | .merge_commit_sha')" || merge=""
  fi
  if [ -z "$merge" ]; then
    echo "::error::go install of $ref failed, and it is not the head of a merged fitness-runner pull request, so no other commit stands in for it."
    exit 1
  fi
  echo "::notice::Pull request $number merged and its head $ref is gone, so fitness builds from its merge commit $merge."
  go install "$module@$merge"
fi
echo "$(go env GOPATH)/bin" >> "$GITHUB_PATH"
