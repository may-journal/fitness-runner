// Command fitness-check-architecture-contract validates declarations beside
// all five architecture views before more specific checks resolve their
// proof. It reads one model from the diagrams and callout tables already in
// each view, and adds no fields or columns. Later checks share the model
// through `internal/architecture.Load` and requirement clauses through
// `internal/requirement.Load`.
//
// # Views
//
// The roots are `architecture/` and `docs/architecture/`. Both roots add to
// one model.
//
//   - `01-system-context.md`: people and systems.
//   - `02-containers.md`: runtime units.
//   - `03-components.md`: components and their sources.
//   - `04-code.md`: classes and functions.
//   - `05-user-journey.md`: journey steps and the requirement IDs they fulfil.
//
// No views means a pass with zero files, unless the repo has product code in
// a [language module]'s sources. Then the check fails and asks for all five.
// Product code is any such tracked file except tests, tool configs, and files
// marked `linguist-generated`. Each module names its own tests and tool
// configs.
//
// Any view requires all five, and each view needs a diagram outside its
// legends. Scoped runs read the whole model.
//
// # Templates
//
// Each view has a template in [internal/architecture/templates], and fitness
// ships all five inside the binary. Start each view the repo lacks as a copy
// of its template:
//
//	mkdir -p docs/architecture
//	for view in 01-system-context 02-containers 03-components 04-code 05-user-journey; do
//	  test -e "docs/architecture/$view.md" || curl -fsSL -o "docs/architecture/$view.md" \
//	    "https://raw.githubusercontent.com/may-journal/fitness-runner/v1/go/internal/architecture/templates/$view.md"
//	done
//
// The `doc-template` check judges every view under `docs/architecture/` or
// `architecture/` against its shipped template. A repo needs no copy of the
// templates; see [shipped templates]. Each template's comments say what to
// write. Each level has its own diagram, kinds, and palette:
//
//   - 01 System context: `C4Context`, with Person, System, and `System_Ext`; System `#1d4ed8`.
//   - 02 Containers: `C4Container`, with Person, `System_Boundary`, Container, and `System_Ext`; Container `#0f766e`.
//   - 03 Components: `C4Component`, with Person, `Container_Boundary`, Component, and `System_Ext`; Component `#b45309`.
//   - 04 Code: `classDiagram`, with classes; class `#6d28d9`.
//   - 05 User journey: `flowchart TB`, with the four journey classes and its `classDef` lines.
//
// People are slate `#334155` and outside systems grey `#64748b` on every
// level, all with white text. Each fill is a fixed color that keeps 4.5:1
// contrast with white, so GitHub's light and dark themes draw it alike.
//
// # What the diagrams already say
//
// Each element a diagram declares carries these facts:
//
//   - Identity: the Mermaid ID, such as `user` in `Person(user, "1 User")`; the same ID in two views is one element.
//   - Kind: the C4 macro, flowchart `:::class`, or `class`.
//   - Name: the label without its callout number.
//   - Parent: the enclosing C4 boundary, subgraph, or namespace.
//   - Callout number: a display reference that binds a table row; renumbering keeps identity.
//
// Its relations and class bodies carry these:
//
//   - A directed relationship: `Rel(...)` or an arrow; left-facing class arrows and `Rel_Back` are reversed.
//   - A relationship kind, kept for the code check: the arrow itself, such as `-->` or `..>`.
//   - A member of the class: a class body line, such as `+Ok` or `+Run(string) error`.
//   - An annotation of the class: a class annotation, such as `<<abstract>>`.
//
// Legends stay out of the model: unnumbered diagrams, C4 diagrams titled
// `Legend`, and diagrams under a `Legend` heading with their tables.
//
// # What callout rows already say
//
//   - A code span or diagram text naming a path under a tracked top-level folder claims that source for the element.
//   - A code span or label naming a tracked root-level source, such as `index.ts`, claims that source.
//   - A trailing `*` in such a path, as in `go/cmd/fitness-check-*`, claims every sibling the pattern matches.
//   - A link to a tracked source file, optionally `#L3` or `#L3-L9`, is static evidence for the element or edge.
//   - A link to `docs/requirements/NNNN-*.md`, or a code span such as `0001.1`, is a requirement reference.
//
// Other text is prose. For example, a consumer's `node_modules/.bin` is no
// claim, since `node_modules` is not a tracked folder.
//
// # Failures
//
//   - Duplicate or invalid IDs, and edges to IDs the diagram does not declare.
//   - A container in a container view, or a component in a component view, outside any boundary.
//   - One person ID with two names, or one person name on two IDs.
//   - Two elements claiming one source path when neither encloses the other.
//   - Claimed paths, evidence files, or line ranges that do not exist.
//   - Requirement references that resolve to no doc or acceptance. Doc shape errors belong to the `requirements` check.
//   - Unsupported Mermaid forms and statements.
//
// Front matter, accessibility lines, C4 `Update…` style macros, and named C4
// arguments such as `$tags` are styling and stay out of the model.
//
// Every error names the view line. An error about two places names both.
//
// # Sources and proof
//
// Every tracked source stays in the inventory as product, test, tool, or
// generated, with the elements whose claims cover it. Tool sources are
// scripts and root-level script tool configs such as `rollup.config.mjs`.
// Unowned sources are listed, not failed; the component check decides
// ownership.
//
// Static evidence only locates a line; later checks resolve what it proves.
// Runtime proof has no form in the diagrams, so no edge counts as proved at
// runtime.
//
// # Requirements
//
// [Requirement 0070] has one installed-command test per acceptance.
//
// [language module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/langs/langs.go
// [internal/architecture/templates]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/architecture/templates
// [shipped templates]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-doc-template/README.md#shipped-templates
// [Requirement 0070]: https://github.com/may-journal/fitness-runner/blob/main/docs/requirements/0070-checks-architecture-contract.md
package check
