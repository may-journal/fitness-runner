// Command fitness-check-pr-structure validates that a pull request
// description follows the [PR template]. It requires a one-line blockquote
// summary, a `## Background` section, and a `## Changelog` section with a
// bullet, and no other `##` sections. The "what changed" lives in the diff,
// so the description carries only the why and the changelog.
//
// A PR description is not a file in the tree, so this check reads its target
// from stdin. It also accepts a `--body-file` path (`-` meaning stdin) and
// falls back to the runner's context-inline `--body` value. With no input at
// all it passes with zero files checked, so the file runner never trips on
// it. It is driven by the [pr-check workflow], not the local suite.
//
// # Behavior
//
//   - Pass: a non-placeholder blockquote summary before the first `##`; a `## Background` section; a `## Changelog` with a `-` bullet; no other `##` sections.
//   - Fail: a missing or placeholder summary, a missing section, a changelog without bullets, or any extra `##` section — each reported separately.
//
// Fenced code is skipped, so a `##` or bullet inside a code sample never
// counts.
//
// The bullet links the PR's [entry file] rather than copying it. This check
// sees only the body, so `pr-check` enforces the link with the PR's diff. It
// fails a PR that adds an entry without linking it, links another file, or
// copies a bullet the PR adds.
//
// [PR template]: https://github.com/may-journal/fitness-runner/blob/main/.github/PULL_REQUEST_TEMPLATE.md
// [pr-check workflow]: https://github.com/may-journal/fitness-runner/blob/main/.github/workflows/README.md
// [entry file]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-changelog/README.md#entry-files
package check
