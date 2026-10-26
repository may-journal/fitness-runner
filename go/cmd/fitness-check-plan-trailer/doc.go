// Command fitness-check-plan-trailer validates an optional `Plan #NN` commit
// trailer. A commit may name the Plan Issue it implements with a line reading
// exactly `Plan #<number>`. The trailer is optional, so a message without
// one passes. A plan reference that is present must be well formed.
//
// A Plan in another repo of the same owner, such as an org-wide Plan in the
// org's `.github` repo, reads `Plan <owner>/<repo>#<number>` or
// `Plan https://github.com/<owner>/<repo>/issues/<number>`. The trailer goes
// on the branch's first commit only, since [issue-link-once] fails a repeat
// in any of the spellings.
//
// Like `semantic-commit`, it reads the proposed message from the
// context-inline `--message` value, falling back to the HEAD commit message.
// The `commit-msg` hook runs it beside `semantic-commit`.
//
// # Behavior
//
//   - Pass: the message has no plan-reference line, or its reference reads exactly one of the three spellings.
//   - Fail: a plan-reference line in another spelling, one error each, with a fix hint naming all three.
//   - `Plan 63`, `plan #63`, `Plan: 63` and `Plan may-journal/.github 63` all fail.
//   - Body prose is never a trailer. A line like "Plan the rollout in 3 steps" is ignored.
//   - `filesChecked` is always 1 (the one commit message).
//
// The pre-push gate accepts a qualified Plan only from the pushing repo's
// own owner, and reads it from the repo it names.
//
// [issue-link-once]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-issue-link-once/README.md
package check
