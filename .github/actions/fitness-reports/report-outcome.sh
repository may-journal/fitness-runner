#!/usr/bin/env bash
# Annotates a failure fitness did not annotate, and links the complete reports.
# Reads JOB_STATUS, ALREADY_ANNOTATED and REPORTS_URL from the environment.
set -eo pipefail
if [ "$JOB_STATUS" = failure ] && [ "$ALREADY_ANNOTATED" != true ]; then
  printf '::error::Workflow job %s failed; open its logs for the failed step and cause.\n' "$GITHUB_JOB"
fi
if [ -n "$REPORTS_URL" ]; then
  printf '[Download complete Fitness reports](%s)\n' "$REPORTS_URL" >> "$GITHUB_STEP_SUMMARY"
fi
