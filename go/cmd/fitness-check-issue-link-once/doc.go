// Command fitness-check-issue-link-once keeps each Issue's timeline readable
// ([#153]). Every commit that links an Issue adds an entry there, so a branch
// links an Issue from one commit at most.
//
// # Behavior
//
//   - A link is `#NN`, `owner/repo#NN`, or an Issue or PR URL on github.com. A bare `#NN` belongs to origin's repo.
//   - Fail: an earlier commit of the branch's own, one no other pushed branch holds, links the same Issue.
//   - A branch stacked on a pushed epic branch so starts where that branch ends.
//   - Fail: the open PR body links the Issue, and the commit is newer than the PR.
//   - Comment lines and anything below the scissors line are ignored, as git drops them.
//
// # Where it runs
//
// The `commit-msg` hook judges the proposed message, which it passes as the
// context-inline `--message` value. An amend leaves out HEAD, the commit it
// replaces. With no open PR, or no answer from gh, it warns and skips the PR
// rule. When GitHub's API host misses the runner's probe, it skips only the
// open PR read, with a note naming the host and proxy.
//
// On a pull request, the org `fitness-suite` workflow judges every branch
// commit, oldest first, in every repo ([#157]). The one link written before
// the PR opened passes, judged by author date, which a rebase keeps. A
// failure stays red until you rewrite that commit without the link and
// force-push with `--force-with-lease`.
//
// Anywhere else, such as a local suite run or a push to `main`, it passes
// inert.
//
// [#153]: https://github.com/may-journal/fitness-runner/issues/153
// [#157]: https://github.com/may-journal/fitness-runner/issues/157
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
