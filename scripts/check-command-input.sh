#!/usr/bin/env bash
# Fails unless FITNESS_COMMAND is issue-check or plan-check, the two commands
# the issue-check workflow runs.
set -e
case "$FITNESS_COMMAND" in
  issue-check|plan-check) ;;
  *) printf '::error::command must be issue-check or plan-check, not %s\n' "$FITNESS_COMMAND"; exit 1 ;;
esac
