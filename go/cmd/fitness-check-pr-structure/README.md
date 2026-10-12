---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# pr-structure

Validates that a pull request description follows the [PR template](../../../.github/PULL_REQUEST_TEMPLATE.md). Required: a one-line blockquote summary, a `## Background` section, a `## Changelog` section that links the PR's `CHANGELOG.md` entry, and no other `##` sections. What changed lives once, in that entry and the diff, so the description carries only the why and the link.

A PR description is not a file in the tree, so this check reads its target from stdin. It also accepts a `--body-file` path (`-` meaning stdin) and falls back to the runner's context-inline `--body` value. With no input at all it passes with zero files checked, so the file runner never trips on it. It is driven by the [pr-check workflow](../../../.github/workflows/README.md), not the local suite.

## Behavior

- Pass: a real blockquote summary first; `## Background`; a `## Changelog` linking a `CHANGELOG.md#anchor` heading; no other `##` sections.
- Fail: a missing or placeholder summary, a missing section, a changelog without that link, or any extra `##` section — each reported separately.

Fenced code is skipped, so a `##` or link inside a code sample never counts.

`pr-check` also reads the PR's `CHANGELOG.md` diff. It fails a link to another heading, and a description that copies a bullet the PR adds. A release PR links its version section instead of a dated entry.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-pr-structure`) that shares its template-validation logic with `plan-structure` through `internal/mdtemplate`. To change the rules, extend that package and the check's tests here and keep this README in sync.
