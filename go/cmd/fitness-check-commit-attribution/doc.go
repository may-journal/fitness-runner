// Command fitness-check-commit-attribution requires a commit message to
// disclose AI use in "AI-Tools:" and "AI-Models:" git trailers. Each must be
// present with a non-empty value on its own line after the subject line.
//
// It applies only to repos that use the convention: the message being
// checked, or one of the last 100 commits, carries either trailer. Elsewhere
// it passes clean with zero files.
//
// # Adopt
//
// Add both trailers to a commit, and every later commit must carry them too.
// Wire the check into a commit-msg hook so a message is validated before the
// commit lands:
//
//	# .git/hooks/commit-msg
//	fitness commit-attribution --message="$(cat "$1")"
//
// # What passes
//
// Both trailers, each on its own line with a non-empty value, appear
// anywhere in the body in any order. Other trailers may sit alongside:
//
//	feat(api): add pagination to the search endpoint
//
//	Support cursor-based paging so large result sets stay fast.
//
//	AI-Tools: Claude Code
//	AI-Models: Opus 4.8
//	Co-Authored-By: A. Dev <a@dev.io>
//
// # What fails
//
// One error per missing trailer. This body has neither:
//
//	feat(api): add pagination
//
//	Just a body, no attribution.
//
// It fails with:
//
//	commit message missing "AI-Models:" trailer
//	commit message missing "AI-Tools:" trailer
//
// A trailer with an empty value does not count as present. An empty message
// (no subject) fails with one guidance error naming both trailers.
//
// # Advanced
//
// Merge and revert commits are exempt — the subject prefix is matched
// literally:
//
//	Merge branch 'feature' into main    → exempt
//	Revert "feat(api): add endpoint"    → exempt
//
// Detection is per-line via `^<Key>:[ \t]*(\S.*)$`, so an `AI-Tools:`
// mid-sentence is not mistaken for a trailer. To change the required keys,
// edit the check and its tests.
//
// # Behavior
//
//   - Declares `--message` as its context-inline argument, mirroring `semantic-commit`.
//   - The runner forwards a proposed message, such as a commit-msg hook's, as `FITNESS_CTX_MESSAGE`; else the check reads the last commit.
//   - A message that is provided but empty fails rather than falling back to `git log`.
//   - Pass: both trailers present with non-empty values, or a `Merge ` or `Revert ` subject.
//   - Fail: one error per missing trailer; on no repo, git error, or empty message, the empty-message guidance error.
//   - `filesChecked` is always 1.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
