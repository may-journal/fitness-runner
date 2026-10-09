#!/bin/sh
# Installs the real tools the requirement tests run, pinned, the way a
# consumer repo has them: eslint and prettier with the shared configs'
# plugins, and vitest with its coverage provider. Run it once from the repo root.
set -eu
cd "$(dirname "$0")"
npm install --no-save --no-audit --no-fund --no-package-lock \
  @typescript-eslint/eslint-plugin@8.71.1 \
  @typescript-eslint/parser@8.71.1 \
  @vitest/coverage-v8@5.0.3 \
  eslint@10.12.0 \
  eslint-config-prettier@10.1.8 \
  eslint-plugin-jsdoc@65.2.2 \
  eslint-plugin-perfectionist@5.12.1 \
  prettier@3.9.9 \
  prettier-plugin-packagejson@3.0.2 \
  prettier-plugin-sort-json@4.2.0 \
  typescript@6.0.3 \
  vitest@5.0.3
