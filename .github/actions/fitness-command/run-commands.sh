#!/usr/bin/env bash
# Runs each fitness command in FITNESS_COMMANDS after checking every name, so a typo runs nothing and one run lists every failure.
set -eo pipefail
# refuse annotates the bad input once and stops before any command.
refuse() {
  printf '::error::The fitness-command action %s.\n' "$1"
  printf 'FITNESS_FAILURE_REPORTED=true\n' >> "$GITHUB_ENV"
  exit 1
}
read -ra commands <<< "$FITNESS_COMMANDS"
if [ "${#commands[@]}" -eq 0 ]; then refuse "needs at least one command"; fi
for command in "${commands[@]}"; do
  if [[ ! "$command" =~ ^[a-z-]+$ ]]; then refuse "got $command, which is not a fitness command name"; fi
done
status=0
for command in "${commands[@]}"; do
  fitness "$command" || status=1
done
exit "$status"
