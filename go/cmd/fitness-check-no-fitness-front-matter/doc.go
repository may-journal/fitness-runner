// Command fitness-check-no-fitness-front-matter fails a Markdown file whose
// front matter still holds `fitnessFunctions` or `relatedConfigurations`.
// No check reads either key, so a leftover block only teaches agents to copy
// it.
//
// # Behavior
//
//   - Pass: no tracked .md file's leading front matter holds either key.
//   - Fail: one error per file holding one, naming the file and telling you to run `fitness strip-front-matter`.
//   - `filesChecked` counts the .md files judged, test fixtures left out.
//
// It reads a leading `---` block, or the same block inside a leading HTML
// comment, as a PR template carries it. A byte order mark or CRLF line
// endings change nothing. Other keys, such as an issue template's `name` and
// `labels`, pass. A leading `---` before prose is a rule, so it passes too.
//
// Test fixtures are out of scope. Files under a `testdata` or
// `tests/examples` folder are never judged, so fixture repos may keep their
// blocks as test data.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
