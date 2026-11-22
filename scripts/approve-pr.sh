#!/usr/bin/env bash
# Approves the PR at the checked commit unless the approver already did, reading GH_TOKEN, PR, SHA and BOT.
set -e
reviews="repos/$GITHUB_REPOSITORY/pulls/$PR/reviews"
approved="$(gh api "$reviews" --paginate |
  jq -rs --arg bot "$BOT" --arg sha "$SHA" 'add | any(.user.login == $bot and .commit_id == $sha and .state == "APPROVED")')"
if [ "$approved" = true ]; then
  echo "PR #$PR is already approved at $SHA."
  exit 0
fi
gh api -X POST "$reviews" -f event=APPROVE -f commit_id="$SHA" \
  -f body="Approved: this PR changes only what Release approve allows for its kind, and its checks are green." > /dev/null
echo "Approved PR #$PR at $SHA."
