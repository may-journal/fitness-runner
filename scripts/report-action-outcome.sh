#!/usr/bin/env bash
# Annotates a failed installation nobody annotated, and writes the action's
# outcome to the job summary. Reads REPORT_STATUS and REPORT_ALREADY_ANNOTATED.
set -eo pipefail
if [ "$REPORT_STATUS" = failure ] && [ "$REPORT_ALREADY_ANNOTATED" != true ]; then
  printf '::error::Fitness installation or execution failed; open the action logs for the failed operation and its cause.\n'
  printf 'FITNESS_FAILURE_REPORTED=true\n' >> "$GITHUB_ENV"
fi
printf '### Fitness action: %s\n\n[View steps and logs](%s/%s/actions/runs/%s)\n' "$REPORT_STATUS" "$GITHUB_SERVER_URL" "$GITHUB_REPOSITORY" "$GITHUB_RUN_ID" >> "$GITHUB_STEP_SUMMARY"
