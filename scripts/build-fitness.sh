#!/usr/bin/env bash
# Builds every fitness command from the checkout into go/bin, so this repo's CI runs the code under test.
set -e
cd go && mkdir -p bin && go build -o bin ./cmd/...
