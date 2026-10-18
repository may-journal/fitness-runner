---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# epic-structure

Validates that an epic follows the [Epic issue template](../../../.github/ISSUE_TEMPLATE/epic.md). An epic is a GitHub Issue body under the `Epic` label. Required: a one-line blockquote pitch, a `## Background` section, and a `## What needs to happen` list of issue references in run order. No other `##` sections are allowed.

Epics live as GitHub Issues, so this check reads its target from stdin (or a `--body-file` path, `-` meaning stdin). It falls back to the runner's context-inline `--body` value. With no input at all it passes with zero files checked. It is driven by `fitness issue-check` in the [issue-check workflow](../../../.github/workflows/README.md), not the local suite.

## Behavior

- Pass: each list item is a plain `#N` or `org/repo#N` reference, with an optional short note after a comma, such as `#12, after #11`.
- Fail: a checkbox, an emoji, a status word such as done or blocked, or an item that is not a plain reference.
- Fail: a reference listed twice, or an item listed before an item its `after` note names.
- Fail: a table anywhere in the body.
- Fail: a list over the prose-budget `maxListItems` cap, 8 by default, unless `###` headings split it into groups within the cap.

Fenced code is skipped, so a `##` or list item inside a code sample never counts.

Checks never call GitHub, so `fitness issue-check` compares the list with the epic's sub-issues. Every listed issue must be a sub-issue and exist, and every sub-issue must be listed.

## Contributing

This README is the canonical description for this check. This check is a self-contained binary (`fitness-check-epic-structure`) that reads the list through `internal/epic` and the sections through `internal/mdtemplate`. To change the rules, extend those packages and the requirement tests, and keep this README in sync.
