#!/usr/bin/env bash
# Links the release PR to its tracking issue, opening the issue when none exists, since every PR must close one.
set -eo pipefail
title="Publish the release from PR #$PR"
issue="$(gh issue list --state all --search "\"$title\" in:title" --json number,title |
  jq -r --arg t "$title" 'map(select(.title == $t))[0].number // empty')"
if [ -z "$issue" ]; then
  text="Release Please owns the version and notes in #$PR."
  url="$(gh issue create --title "$title" --body "$text")"
  issue="${url##*/}"
fi
body="$(gh pr view "$PR" --json body --jq .body)"
case "$body" in
  *"Closes #$issue."*) ;;
  *) gh pr edit "$PR" --body "$body

Closes #$issue."
     echo "edited=true" >> "$GITHUB_OUTPUT" ;;
esac
