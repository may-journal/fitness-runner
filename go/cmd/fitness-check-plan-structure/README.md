---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# plan-structure

Validates that a plan — a GitHub Issue body under the `Plan` label — follows the [Plan issue template](../../../.github/ISSUE_TEMPLATE/plan.md): a one-line blockquote pitch, a `## Background` section, and a `## What needs to happen` section with at least one checklist item, and no other `##` sections.

Plans no longer live as files in the repo, so this check reads its target from stdin (or a `--body-file` path, `-` meaning stdin), falling back to the runner's context-inline `--body` value. With no input at all it passes with zero files checked, so the file runner never trips on it. It is driven by the [plan-check workflow](../../../.github/workflows/README.md), not the local suite.

## Behavior

- Pass: one non-placeholder blockquote pitch before the first `##`; a `## Background` section; a `## What needs to happen` section with at least one `- [ ]` or `- [x]` item; no other `##` sections.
- Fail: a missing pitch, a placeholder pitch, a missing section, an empty checklist, or any extra `##` section, each reported as its own error.
- Fail (pointed): an `## Open questions` section reports a message to turn unknowns into checklist steps instead.

Fenced code is skipped, so a `##` or checkbox inside a code sample never counts.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-plan-structure`). To change the rules, extend the check and its tests here and keep this README in sync.
