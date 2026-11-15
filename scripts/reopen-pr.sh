#!/usr/bin/env bash
# Closes and reopens the PR in PR, so pr-check reads its edited body.
set -e
gh pr close "$PR"
gh pr reopen "$PR"
