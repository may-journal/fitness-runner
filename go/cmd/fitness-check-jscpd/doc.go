// Command fitness-check-jscpd fails a repo whose duplicated lines pass 1% of
// its scanned lines, with [jscpd] semantics. A native Go clone-detection
// engine runs instead of the jscpd tool: a generic lexer plus a rolling-hash
// detector in `go/internal/clonedetect`. Its verdict and error format match
// jscpd's; exact percentage parity with jscpd's per-language tokenizers is
// not the goal. It is on by default.
//
// # Behavior
//
//   - Scans every tracked file with min-lines 5, min-tokens 50, tests, Markdown, and JSON included.
//   - Binary files are skipped, since they have no lines to compare.
//   - Files the repo's `.gitattributes` marks `linguist-generated` are left out.
//   - Lockfiles and build wrapper files are left out, as each [language module] or the [lockfiles] folder names them.
//   - Pass: duplicated lines stay under 1% of the codebase, with jscpd's threshold semantics.
//   - Fail: over threshold — the check reports `ERROR: jscpd found too many duplicates (X%) over threshold (Y%)`, then `path: N duplicated lines` for each file holding them.
//   - The engine is built into the check binary — no external jscpd tool to install.
//
// # Scope
//
// jscpd compares files of one format. So a scoped run, such as a commit,
// reads the changed files and the tracked files sharing their endings. A
// file's ending is its extension, or its whole name when it has none.
//
// An unscoped run reads every tracked file. The 1% threshold applies to the
// lines read, so a scoped verdict can be stricter than an unscoped one.
//
// # Templates
//
// A doc must repeat its template, so templates are left out. Any file
// [doc-template] treats as a template is skipped, in any folder. Docs built
// from one template are still compared with each other and with every other
// file.
//
// # Escape hatch
//
// For duplication that's a real, justified constraint, such as boilerplate a
// framework requires per type, wrap it in jscpd's inline-comment markers.
// Don't force a refactor:
//
//	// jscpd:ignore-start
//	...
//	// jscpd:ignore-end
//
// [jscpd]: https://github.com/kucherenko/jscpd
// [language module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/langs/langs.go
// [lockfiles]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/lockfiles
// [doc-template]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-doc-template/README.md#what-is-a-template
package check
