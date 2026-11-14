// Command fitness-check-epic-structure validates that an epic follows the
// [Epic issue template], or the repo's own template labeled `Epic`. An epic
// is a GitHub Issue body under the `Epic` label. Required: a one-line
// blockquote pitch, a `## Background` section, and a `## What needs to happen`
// list of issue references in run order. No other `##` sections are allowed.
//
// Epics live as GitHub Issues, so this check reads its target from stdin (or
// a `--body-file` path, `-` meaning stdin). It falls back to the runner's
// context-inline `--body` value. With no input at all it passes with zero
// files checked. It is driven by `fitness issue-check` in the
// [issue-check workflow], not the local suite.
//
// # Behavior
//
//   - Pass: a `- [ ]` or `- [x]` item with text, its Plan as a full issue URL, then optional pull request URLs.
//   - Pass: a plain `#N` or `org/repo#N` reference, with an optional short note after a comma, such as `#12, after #11`.
//   - Fail: a checklist item with no issue URL, an emoji, or a plain item that is no reference or notes a status.
//   - Fail: a Plan listed twice, or a plain item listed before an item its `after` note names.
//   - Fail: a table anywhere in the body.
//   - A list over the `maxListItems` cap fails prose-budget, which issue-check runs on every body; `###` headings split it into shorter lists.
//
// An item's Plan is the first issue URL in it, and a URL into a comment is
// not an issue. Plans may live in any repository, `.github` included.
// Fenced code is skipped, so a `##` or list item inside a code sample never
// counts. A bare `#N` is an issue in the judged repo, so `#12` and
// `org/repo#12` match.
//
// Checks never call GitHub, so `fitness issue-check` reads each listed issue
// and fails one that is missing or unreadable. It does not read sub-issues.
//
// [Epic issue template]: https://github.com/may-journal/fitness-runner/blob/main/.github/ISSUE_TEMPLATE/epic.md
// [issue-check workflow]: https://github.com/may-journal/fitness-runner/blob/main/.github/workflows/README.md
package check
