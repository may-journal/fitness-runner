// Command fitness-check-plan-structure validates that a plan, a GitHub Issue
// body under the `Plan` label, follows the [Plan issue template]. It requires
// a one-line blockquote pitch, a `## Background` section, and a
// `## What needs to happen` section with at least one checklist item. No
// other `##` sections are allowed.
//
// A repo's own `.github/ISSUE_TEMPLATE` file labeled `Plan` replaces the
// sections below. A section whose template shows a checklist item needs a
// checklist item.
//
// Plans no longer live as repo files, so this check reads its target from
// stdin (or a `--body-file` path, `-` meaning stdin). It falls back to the
// runner's context-inline `--body` value. With no input at all it passes with
// zero files checked, so the file runner never trips on it. It is driven by
// the [issue-check workflow], not the local suite.
//
// # Behavior
//
//   - Pass: a non-placeholder blockquote pitch before the first `##`, and a `## Background` section.
//   - Pass: a `## What needs to happen` section with a `- [ ]` or `- [x]` item, and no other `##` sections.
//   - Fail: a missing pitch, a placeholder pitch, a missing section, an empty checklist, or any extra `##` section — each its own error.
//   - Fail (pointed): an `## Open questions` section reports a message to turn unknowns into checklist steps instead.
//
// Fenced code is skipped, so a `##` or checkbox inside a code sample never
// counts.
//
// [Plan issue template]: https://github.com/may-journal/fitness-runner/blob/main/.github/ISSUE_TEMPLATE/plan.md
// [issue-check workflow]: https://github.com/may-journal/fitness-runner/blob/main/.github/workflows/README.md
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
