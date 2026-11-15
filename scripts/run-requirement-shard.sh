#!/usr/bin/env bash
# Runs one shard of the requirement tests with the candidate Go installer.
# Reads PLATFORM and SHARD, which names the shard of two.
set -e
chmod +x out/fitness-install-*-"$PLATFORM"
FITNESS_INSTALL="$(ls "$PWD"/out/fitness-install-*-"$PLATFORM")"
export FITNESS_INSTALL
sh tests/requirements-shard.sh "$SHARD" 2
