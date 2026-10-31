// Command fitness-check-mermaid-level-bleed warns when a callout description
// at one C4 level repeats the previous level verbatim. Lower levels should
// add level-specific rationale, not restate the parent. It is part of the
// mermaid diagram and callout table fitness-function set ([#28]).
//
// # Behavior
//
// Across numbered `architecture/NN-*.md` files, it compares callout
// descriptions at adjacent levels. It warns when a lower level repeats an
// upper level verbatim, adding nothing new. It compares only exact,
// normalized matches, to stay low on false positives.
//
// `filesChecked` counts the numbered level files, whether or not they
// contain mermaid blocks.
//
// [#28]: https://github.com/may-journal/fitness-runner/issues/28
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
