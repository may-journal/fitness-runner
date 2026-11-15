#!/usr/bin/env bash
# Checks the approver token in TOKEN for the KIND of approval, writing ready and login when it is usable.
set -e
case "$KIND" in
  release | dependency) ;;
  *) echo "::error::Release approve got kind '$KIND'; pass release or dependency."; exit 1 ;;
esac
if [ -z "$TOKEN" ]; then
  echo "::notice::APPROVER_TOKEN is not available to this repo, so a person approves the PR."
  exit 0
fi
error="$(mktemp)"
if login="$(GH_TOKEN="$TOKEN" gh api user --jq .login 2> "$error")" &&
  GH_TOKEN="$TOKEN" gh api "repos/$GITHUB_REPOSITORY" --silent 2> "$error"; then
  printf 'ready=true\nlogin=%s\n' "$login" >> "$GITHUB_OUTPUT"
  exit 0
fi
case "$(cat "$error")" in
  *"HTTP 401"*) echo "::error::GitHub rejects APPROVER_TOKEN; replace the org secret with a valid token." ;;
  *"HTTP 404"*) echo "::error::APPROVER_TOKEN cannot see $GITHUB_REPOSITORY; give the token access to it." ;;
  *) echo "::error::Could not check APPROVER_TOKEN: $(cat "$error")" ;;
esac
exit 1
