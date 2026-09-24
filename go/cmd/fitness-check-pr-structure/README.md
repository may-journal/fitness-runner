---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# pr-structure

Validates that a pull request description follows the [PR template](../../../.github/PULL_REQUEST_TEMPLATE.md). Required: a one-line blockquote summary, a `## Background` section, a `## Changelog` section with at least one bullet, and no other `##` sections. The "what changed" lives in the diff, so the description carries only the why and the changelog.

A PR description is not a file in the tree, so this check reads its target from stdin (or a `--body-file` path, `-` meaning stdin). It falls back to the runner's context-inline `--body` value. With no input at all it passes with zero files checked, so the file runner never trips on it. It is driven by the [pr-check workflow](../../../.github/workflows/README.md), not the local suite.

## Behavior

- Pass: one non-placeholder blockquote summary before the first `##`; a `## Background` section; a `## Changelog` section with at least one `-` bullet; no other `##` sections.
- Fail: a missing summary, a placeholder summary, a missing section, a changelog with no bullets, or any extra `##` section, each reported as its own error.

Fenced code is skipped, so a `##` or bullet inside a code sample never counts.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-pr-structure`) that shares its template-validation logic with `plan-structure` through `internal/mdtemplate`. To change the rules, extend that package and the check's tests here and keep this README in sync.
