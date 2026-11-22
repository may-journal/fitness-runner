#!/usr/bin/env bash
# Fails with the secret's name when the automation App's id or key did not
# reach this repo. Reads APP_ID and APP_KEY from the environment.
set -eo pipefail
for name in APP_ID APP_KEY; do
  if [ -z "${!name}" ]; then
    echo "::error::AUTOMATION_${name} is not available to this repo"
    exit 1
  fi
done
