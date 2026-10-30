// Command fitness-check-mermaid-callouts requires every mermaid diagram with
// callouts to have a callout table whose callouts match it one to one.
// Orphan callouts, orphan rows, and duplicates fail. It is the foundational
// check of the mermaid diagram and callout table fitness-function set
// ([#28]).
//
// # Behavior
//
// For each `.md` file, diagrams (fenced `mermaid` blocks) are paired with the
// callout table that follows them. That table is a GFM table whose first
// header cell is `#`, `No`, `Callout`, or `Ref`.
//
// Nodes carry number callouts and connectors carry letter callouts, so a
// reader tells them apart at a glance ([#583]). Letters run `A` to `Z`, then
// double: `AA`, `BB`, up to `ZZ`, then `AAA`. The table's `#` column holds the
// same letters for connector rows.
//
//   - Pass: every callout in the diagram maps to exactly one table row, and vice versa. Diagrams without callouts need no table.
//   - Fail (orphan callout): `diagram callout N has no matching table row`.
//   - Fail (orphan row): `callout table row N has no matching diagram callout`.
//   - Fail (duplicate): `diagram callout N appears K times` or `callout table row N appears K times`.
//   - Fail (numbered connector): `connector callout N must be a letter; numbers are for nodes; run fitness mermaid-callouts --fix`.
//   - Fail (broken doubling): `connector callout AB breaks the letter doubling`, or the same for a table row. A label such as YES that no row lists is prose.
//   - Fail (unpaired diagram): `mermaid diagram … has callouts but no associated callout table`.
//   - Fail (unpaired table): `callout table … has no preceding mermaid diagram`.
//
// # Reading callouts
//
// A node callout is the leading integer of a node label: quoted
// (`Container(x, "3", …)`, `A["1 Check"]`) or inside a node shape
// (`A[2 Screen]`).
//
// A connector callout is the leading label of an edge, in any of these forms:
//
//   - C4 relationship: `Rel(a, b, "A")`.
//   - Flowchart pipe: `a -->|B| b` or `a -->|"B"| b`.
//   - Inline link: `a -- C --> b`.
//   - Colon after an arrow: `a --> b : D` or `a ->> b: D`.
//
// A colon label with no arrow before it, such as a state description or a
// note, is a node's. Capital letters that break the doubling count only when
// they are the whole label, so prose such as `|API calls|` is not a callout.
// Such a label also never makes a diagram need a table on its own. In the
// table, callouts come from the first (`#`) column: a number, capital
// letters, or either in brackets.
//
// # Fix
//
// `fitness mermaid-callouts --fix` converts a repo's numbered connectors
// ([#702]), then judges the result as a plain run does. It reads each
// diagram with the parser the check uses, so it converts exactly what the
// check fails.
//
//   - Convert: each numbered connector becomes the next letter its diagram and table do not use yet. Its row's `#` cell follows.
//   - Keep: node numbers, other labels, and every other byte stay as they were.
//   - Report: one line per file it changed, naming each conversion, then the count.
//   - Skip: a number that also marks a node or two connectors, or has no single row, leaves its diagram unchanged.
//   - Name: each skipped diagram gets a line with its line number and why.
//   - Repeat: a second run changes nothing.
//
// A skipped diagram still fails, so fix it by hand.
//
// # Scope
//
// Runs on every `.md` file that contains a diagram or callout table. Files
// with neither are skipped and don't count toward `filesChecked`.
//
// [#28]: https://github.com/may-journal/fitness-runner/issues/28
// [#583]: https://github.com/may-journal/fitness-runner/issues/583
// [#702]: https://github.com/may-journal/fitness-runner/issues/702
package check
