#!/usr/bin/env bash
# Shapes the release PR's notes for pr-check, with plain bullets and a compare link that points at the head commit.
set -eo pipefail
body="$(gh pr view "$PR" --json body --jq .body)"
head="$(gh pr view "$PR" --json headRefOid --jq .headRefOid)"
shaped="$(printf '%s\n' "$body" | awk -v head="$head" '
  after && first { first = 0; if ($0 != "") print "" }
  { line = $0 }
  after && /^##+ \[.*\/compare\// { sub(/\.\.\.[^)]*\)/, "..." head ")", line) }
  after { sub(/^## /, "### ", line); while (sub(/\*\*/, "", line)) {} }
  { print line }
  $0 == "## Changelog" { after = 1; first = 1 }')"
if [ "$shaped" != "$body" ]; then
  gh pr edit "$PR" --body "$shaped"
  echo "edited=true" >> "$GITHUB_OUTPUT"
fi
