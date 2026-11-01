// Command fitness-check-duplicate-code fails a repo whose duplicated lines
// pass 1% of its scanned lines. A native Go clone-detection engine judges
// them: a generic lexer plus a rolling-hash detector in
// `go/internal/clonedetect`. Its thresholds are modeled on [jscpd]'s, but
// no jscpd tool runs, and percentage parity with it is not the goal. It is
// on by default.
//
// # Behavior
//
//   - Scans every tracked file with min-lines 5, min-tokens 50, tests, Markdown, and JSON included.
//   - Binary files are skipped, since they have no lines to compare.
//   - Files the repo's `.gitattributes` marks `linguist-generated` are left out.
//   - Lockfiles and build wrapper files are left out, as each [language module] or the [lockfiles] folder names them.
//   - Pass: duplicated lines stay at or under 1% of the scanned lines.
//   - Fail: over threshold — the check reports `ERROR: duplicate code is X% of scanned lines, over the Y% threshold`, then `path: N duplicated lines` for each file holding them.
//   - The engine is built into the check binary, so there is no external tool to install.
//
// # Scope
//
// The check compares files of one format, so a scoped run reads the changed
// files and tracked files sharing their endings. A
// file's ending is its extension, or its whole name when it has none.
//
// An unscoped run reads every tracked file. The 1% threshold applies to the
// lines read, so a scoped verdict can be stricter than an unscoped one.
//
// # Comments
//
// Comments never count as duplication. The lexer drops each comment, from
// `//`, `#`, `/*` or `<!--` to its end, before it compares tokens. Two files
// sharing only identical comments pass, as a real run in requirement 0139
// shows. Only the ignore markers below act, as directives that count as code.
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
// framework requires per type, wrap it in inline-comment markers. Don't
// force a refactor:
//
//	// duplicate-code:ignore-start
//	...
//	// duplicate-code:ignore-end
//
// The older `jscpd:ignore-start` and `jscpd:ignore-end` markers still work
// as deprecated aliases, so repos that carry them keep passing.
//
// [jscpd]: https://github.com/kucherenko/jscpd
// [language module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/langs/langs.go
// [lockfiles]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/lockfiles
// [doc-template]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-doc-template/README.md#what-is-a-template
package check
