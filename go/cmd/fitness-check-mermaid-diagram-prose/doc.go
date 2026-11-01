// Command fitness-check-mermaid-diagram-prose fails edge labels that carry
// prose beyond a callout, number or letter, in a diagram paired with a
// callout table. Diagram labels must not duplicate callout table prose. It is
// part of the mermaid diagram and callout table fitness-function set ([#28]).
//
// # Behavior
//
// In a diagram paired with a callout table, it flags edge labels with prose
// after a callout number or letter. For example, `"6 Uses"` should be `"6"`
// and `"A calls"` should be `"A"`. Arrows carry callouts only; descriptions
// belong in the table, not the diagram.
//
// It runs on every `.md` file with a mermaid diagram or callout table; files
// with neither are skipped and don't count toward `filesChecked`.
//
// [#28]: https://github.com/may-journal/fitness-runner/issues/28
package check
