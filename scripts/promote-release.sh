#!/usr/bin/env bash
# Promotes the verified release with the Linux release tool, using the automation App's token in GH_TOKEN.
set -e
chmod +x out/fitness-release-linux-amd64
out/fitness-release-linux-amd64 promote
