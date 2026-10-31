// Command fitness-check-issue-checklist fails an issue body whose task list
// still has unchecked items, so closing it would close unfinished work.
//
// It reads the body from a `--body-file` path, `-` meaning stdin, or from the
// runner's context-inline `--body` value. With no input it passes with zero
// files checked. `fitness close-check` runs it on a closed issue, and
// `fitness pr-check` on each issue a pull request closes.
//
// # Behavior
//
//   - Pass: every `- [ ]` item is ticked, or the body has no task list.
//   - Fail: one error per unchecked item, quoting its text.
//
// Items inside fenced code are examples, so they never count.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
