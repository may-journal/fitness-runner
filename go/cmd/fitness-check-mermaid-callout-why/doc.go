// Command fitness-check-mermaid-callout-why requires each numbered callout
// table to carry a "Why" column, and flags numbered rows whose "Why" cell is
// empty. So tables explain why each element exists, not just what it is. It
// is part of the mermaid diagram and callout table fitness-function set
// ([#28]).
//
// # Behavior
//
// For each numbered callout table, it requires a `Why` column
// (case-insensitive) in the header. It warns when a callout row leaves its
// `Why` cell empty. A callout row opens with a node number, such as `1`, or a
// connector letter, such as `A`, as [mermaid-callouts] reads them.
//
// It runs on every `.md` file with a mermaid diagram or callout table; files
// with neither are skipped and don't count toward `filesChecked`.
//
// [#28]: https://github.com/may-journal/fitness-runner/issues/28
// [mermaid-callouts]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-mermaid-callouts/README.md
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
