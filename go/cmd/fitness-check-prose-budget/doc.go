// Command fitness-check-prose-budget caps how much prose a markdown file
// holds. It is the volume sibling of [text-readability], which scores
// density. Where that check asks whether prose is dense, this one asks
// whether there is too much of it.
//
// Each `.md` file is masked the same way `text-readability` masks it. Front
// matter, fenced code, tables, headings, and dash-only lines such as a
// Measurement's `-` are dropped. Only flowing paragraphs, blockquotes, and
// list items remain. The check then enforces six
// hard limits on what is left.
//
// # Limits
//
//   - Words per sentence: 23.
//   - Sentences per paragraph: 4.
//   - Paragraphs per section: 3.
//   - Words per list item: 23.
//   - Items per list: 8.
//   - Prose words per section: 300.
//
// Each limit is overridable in `.fitnessrc.json` under `proseBudget`, as the
// configuration below lists. A positive value replaces the default, and zero
// or absent keeps it.
//
// A paragraph is a run of consecutive prose lines. A section runs from one
// heading to the next, and its paragraphs and lists count toward `maxWords`.
// Prose before the first heading is its own section. A list is a run of
// consecutive items; nested items count individually.
//
// # Behavior
//
//   - Pass: every masked paragraph, section, and list in the file is within all six limits.
//   - Fail: one error per limit crossed, naming the file, the offending element, and its count versus the limit.
//   - Every tracked `.md` file is judged, `CHANGELOG.md` included; there is no exempt list.
//   - A `.fitnessrc.json` that sets `proseBudget.exempt` fails the run.
//
// It shares its prose extraction and sentence splitting with
// `text-readability` through `internal/mdx`.
//
// # Code comments
//
// The check also judges each code comment block that `fitness comments`
// lists, by the same limits except the section ones. A comment has no
// headings, so it holds no sections. Its markers are removed first, and an
// indented line that carries on no list item is code, which is skipped.
//
// A commit or pull request run judges only the blocks that touch a changed
// line. A full scan, such as `fitness --all prose-budget`, judges every
// block. Each error names the block as `path:line`, the line it starts on.
// Tool directives, license headers and generated files are never judged.
//
// [text-readability]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-text-readability/README.md
package check
