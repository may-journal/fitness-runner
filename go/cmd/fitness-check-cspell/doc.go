// Command fitness-check-cspell fails unknown words in the checked files. It
// spell-checks with a native Go engine and dictionaries embedded in the check
// binary — no external cspell tool, no npm install. The project `cspell.json`
// words always win over the embedded base dictionaries; it ignores no path.
//
// # Behavior
//
//   - Pass: no unknown words in checked files.
//   - In a scoped run: only the changed paths that still exist on disk are checked; with none left it passes without scanning.
//   - Otherwise: spell-checks every tracked file; binary files, with a NUL in their first 8 KB, count but are never spelled or read whole.
//   - Generated files: files `.gitattributes` marks `linguist-generated`, and the [lockfiles and build wrapper files] of any language, are left out; nothing else is.
//   - Fail: a resolved `cspell.json` that sets `ignorePaths` fails before any scan; remove it and fix the words instead.
//   - Config: a repo-local `cspell.json` wins; otherwise an installed `@mayjournal/fitness-shared` package under `node_modules`; otherwise the copy embedded in the binary, materialized on demand.
//   - No copy is required in the consumer repo, so a repo with no npm anywhere still gets the shared words.
//   - Forbidden words: each `flagWords` word or phrase fails wherever it appears whole, in any case, except in a `cspell.json` file.
//
// Errors are reported as one line per issue:
// `path:line:col - Unknown word (word)` or
// `path:line:col - Forbidden word (entry)`.
//
// A file over 64 KB is checked in line-aligned chunks in parallel, once its
// `cspell:` directives are applied to the whole file.
//
// [lockfiles and build wrapper files]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/lang.go
package check
