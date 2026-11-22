#!/usr/bin/env bash
# Writes the JDK version file to the step output, where a .sdkmanrc or .tool-versions counts only with a java line.
# cspell:ignore sdkmanrc
set -eo pipefail
if [ -f .java-version ]; then file=.java-version
elif grep -qs '^java[[:space:]]*=' .sdkmanrc; then file=.sdkmanrc
elif grep -qs '^java[[:space:]]' .tool-versions; then file=.tool-versions
fi
echo "file=${file:-}" >> "$GITHUB_OUTPUT"
