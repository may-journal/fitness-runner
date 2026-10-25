// Command fitness-check-changelog-bullets keeps changelog entries tight:
// every [entry file]'s `###` section must have 3 to 5 bullets, each under 365
// characters. Each bullet starts with a semantic type, as `Feat:` or
// `feat(scope):`.
//
// # Behavior
//
//   - Every entry file is judged, and errors name that file.
//   - A pull request adds one dated entry, and later commits amend it, so its 3 to 5 bullets cover the whole PR.
//   - Bullet count must be 3 to 5, counting only `- ` bullets; continuation lines join their bullet for the length count.
//   - Every bullet must start with one of: `Feat:`, `Fix:`, `Docs:`, `Style:`, `Refactor:`, `Perf:`, `Test:`, `Build:`, `Ci:`, `Chore:`, `Revert:`.
//   - A semantic commit subject works too: a lowercase type with an optional scope and `!`, such as `feat(release): ` or `fix!: `.
//   - `CHANGELOG.md` is not judged, so a repo with no entry files passes with zero files checked.
//
// # Errors
//
//	changelog/2026-05-01-0000-readme.md section "2026.05.01.0000" has 1 bullets; keep entries to 3-5 bullets
//	changelog/2026-05-01-0000-readme.md:7: bullet is 1230 characters; keep each under 365
//	changelog/2026-05-01-0000-readme.md:8: bullet must start with a semantic type (Feat:, Fix:, ...)
//
// [entry file]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-changelog/README.md#entry-files
package check
