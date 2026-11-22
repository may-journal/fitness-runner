#!/usr/bin/env bash
# Writes to the step output whether the release for RELEASE_TAG is still a draft, and fails on any other answer.
set -e
draft="$(gh release view "$RELEASE_TAG" --json isDraft --jq .isDraft)"
case "$draft" in true|false) ;; *) exit 1 ;; esac
printf 'draft=%s\n' "$draft" >> "$GITHUB_OUTPUT"
