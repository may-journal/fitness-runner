#!/usr/bin/env bash
# Writes the Go build cache's prefix, key and save flag to the step outputs.
# Reads FAMILY, GO_VERSION, SOURCE_HASH and SAVE from the environment.
set -eo pipefail
prefix="go-build-$FAMILY-$RUNNER_OS-$RUNNER_ARCH-go$GO_VERSION-"
{
  echo "prefix=$prefix"
  echo "key=$prefix$SOURCE_HASH"
  echo "save=$SAVE"
} >> "$GITHUB_OUTPUT"
