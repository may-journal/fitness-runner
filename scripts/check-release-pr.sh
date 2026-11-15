#!/usr/bin/env bash
# Passes a release PR the automation App opened that changes only release files, and writes the commit it checked as sha.
set -e
view="$(gh pr view "$PR" --json author,state,isDraft,headRefOid)"
author="$(jq -r .author.login <<< "$view")"
if [ "$author" != "$AUTHOR" ]; then
  echo "::notice::PR #$PR is by $author, not $AUTHOR, so a person approves it."
  exit 0
fi
if [ "$(jq -r '.state + " " + (.isDraft | tostring)' <<< "$view")" != "OPEN false" ]; then
  echo "::notice::PR #$PR is not open and ready, so it gets no approval."
  exit 0
fi
files="$(gh api "repos/$GITHUB_REPOSITORY/pulls/$PR/files" --paginate |
  jq -rs 'add | .[] | .filename, (.previous_filename // empty)')"
others="$(grep -vxF -e CHANGELOG.md -e version.txt -e .release-please-manifest.json <<< "$files" || true)"
if [ -z "$files" ] || [ -n "$others" ]; then
  echo "::notice::PR #$PR changes more than the release files, so a person approves it: ${others//$'\n'/, }"
  exit 0
fi
printf 'sha=%s\n' "$(jq -r .headRefOid <<< "$view")" >> "$GITHUB_OUTPUT"
