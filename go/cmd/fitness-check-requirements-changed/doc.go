// Command fitness-check-requirements-changed fails a change that adds or
// changes product code without also changing a test and a requirement doc.
// The [requirements] check judges the docs and tests that exist; this check
// judges what a commit or pull request leaves out.
//
// # What counts
//
//   - Product code: `.go` and `.swift` files that are not tests, and `.sh` scripts.
//   - Vendored code is not product code: files under `vendor`, `third_party`, `Pods`, `Carthage`, or `node_modules`.
//   - Fixtures and build output are not either: Go `testdata`, `dist`, `.build`, `DerivedData`, and what each [language module] names.
//   - Generated files are not: those `.gitattributes` marks `linguist-generated`, and Go files that say `Code generated … DO NOT EDIT.`
//   - Tests: `_test.go` files, and Swift files that declare a test or sit in a `…Tests` folder, as the requirements check pairs them.
//   - Requirement docs: `docs/requirements/NNNN-*.md`.
//
// A file counts as changed only when its lines change beyond whitespace, so
// a pure rename or a reformat changes nothing. Nor does a file whose code,
// with comments removed, is the same before and after. That holds for tests
// and docs too. Directives, a shebang, a cgo preamble, and a Go example's
// output comment count as code.
//
// # Passes
//
//   - Product code changes beside a changed test and a changed requirement doc.
//   - No product code changed, as in a docs-only, config-only, or `CHANGELOG.md`-only change.
//   - A pure rename or a whitespace-only edit.
//   - A comment-only edit, and the spacing it moves.
//   - A local merge, cherry-pick, or revert, or an amended merge, is not judged again. Its pull request diff still is.
//   - A repo with no `docs/requirements/NNNN-*.md` doc yet.
//   - A full scan, or a `--dir` folder, judges nothing.
//   - A deletion of product code that no requirement doc names.
//
// # Fails
//
//   - Product code changed without a test, a requirement doc, or both. The error names the file and the missing half.
//   - Product code deleted while a requirement doc still names it. The error names the file, the doc, and the doc's tests.
//
// A deletion passes when that doc or one of its tests changes with it.
//
// # No exceptions
//
// No commit trailer, flag or setting turns the check off for one change.
//
// # When it runs
//
//   - Commits: the staged files. With a `commit-msg` hook, the hook judges them.
//   - A `commit-msg` hook, as `fitness init` installs, also makes `pre-commit` leave the check out.
//   - Amends: everything the amended commit changes since its parent.
//   - Pull requests: CI judges the diff against the merge base with the base branch.
//
// [requirements]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-requirements/README.md
// [language module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/langs/langs.go
package check
