---
name: vitest-coverage-exclude
description: Fix a failing vitest-coverage-exclude check on a Vitest config. Use when coverage.exclude has entries.
---

# Empty the Vitest coverage exclude

Vitest coverage judges every file, so `coverage.exclude` is empty or missing. [The check README](../../README.md) names the config files it reads.

## Read the failure

- `Vitest coverage exclude must be empty, so coverage judges every file; remove: "src/legacy/**"`: one line for each entry. Remove the entry from `vitest.config.*` or the `vitest` field in `package.json`.

Then run the coverage and see which files now fall short. Add tests that cover those files' behavior. Use `coverage.exclude: []` to measure tests and declarations too.

See [vitest-coverage-full](../../../fitness-check-vitest-coverage-full/skills/vitest-coverage-full/SKILL.md) for the 100% requirement that follows.

## Check it first

Run `fitness vitest-coverage-exclude` from the repo root. Exit 0 means the exclude list is empty.
