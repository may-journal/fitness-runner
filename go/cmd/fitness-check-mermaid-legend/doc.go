// Command fitness-check-mermaid-legend requires numbered flowchart callout
// nodes to carry a style class, and diagrams with callouts to declare a
// `classDef` legend. It is part of the mermaid diagram and callout table
// fitness-function set ([#28]).
//
// # Behavior
//
// A class is inline `:::class` or a `class` or `style` statement. The check
// flags numbered callout nodes with no styling, since consistent styling
// keeps callouts legible in a legend. C4 and other diagram types, with their
// different styling model, are skipped.
//
// Connectors carry letter callouts, not numbers, as [mermaid-callouts]
// requires. Links take `linkStyle`, not a class, so this check skips
// connector letters.
//
// It runs on every `.md` file with a mermaid diagram or callout table; files
// with neither are skipped and don't count toward `filesChecked`.
//
// [#28]: https://github.com/may-journal/fitness-runner/issues/28
// [mermaid-callouts]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-mermaid-callouts/README.md
package check
