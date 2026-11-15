#!/usr/bin/env bash
# Waits for every check but this run's own to pass, and writes green when the PR is ready to approve.
set -e
verdict=pending
error="$(mktemp)"
own="/actions/runs/$GITHUB_RUN_ID/"
for ((try = 1; try <= TRIES; try++)); do
  if [ "$try" -gt 1 ]; then sleep "$WAIT_SECONDS"; fi
  # gh exits 1 on a failed check and 8 on a pending one, and
  # prints the list either way.
  checks="$(gh pr checks "$PR" --json bucket,link 2> "$error")" || true
  if ! jq -e 'type == "array"' <<< "$checks" > /dev/null 2>&1; then
    if grep -q 'no checks reported' "$error"; then continue; fi
    echo "::error::Could not read the checks of PR #$PR: $(cat "$error")"
    exit 1
  fi
  verdict="$(jq -r --arg own "$own" '[.[] | select((.link // "") | contains($own) | not) | .bucket] |
    if any(. == "fail" or . == "cancel") then "red"
    elif length == 0 or any(. == "pending") then "pending" else "green" end' <<< "$checks")"
  if [ "$verdict" != pending ]; then break; fi
done
case "$verdict" in
  red)
    echo "::notice::PR #$PR has a failed or cancelled check, so it gets no approval."
    exit 0 ;;
  pending)
    echo "::warning::PR #$PR still had checks running after $TRIES tries, so a person approves it."
    exit 0 ;;
esac
head="$(gh pr view "$PR" --json headRefOid --jq .headRefOid)"
if [ "$head" != "$SHA" ]; then
  echo "::notice::PR #$PR moved to $head while its checks ran, so this run approves nothing."
  exit 0
fi
echo "green=true" >> "$GITHUB_OUTPUT"
