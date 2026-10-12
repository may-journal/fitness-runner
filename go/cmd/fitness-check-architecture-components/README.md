---
fitnessFunctions: ['architecture-components']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-components

Gives every product source one owner among the components the components view draws. It keeps the references between those components, and the runtimes that reach them, to what the view draws. It reads the model `architecture-contract` describes and adds no fields or columns.

## Owners

A component owns the product sources its claims cover: a path in its diagram strings or its callout row. Boundaries, people, and external systems own nothing; their claims only scope the components inside them. Product sources are the product category of the `architecture-contract` inventory, so tests, tools, and generated files need no owner.

- A product source no component names fails, naming the source.
- A source two components name, in two diagrams of one view, fails naming both claims. Two claims in one diagram fail in `architecture-contract`.
- A claimed path that no longer exists fails in `architecture-contract`.

## References

Every reference between two owners is an observed edge. The view must draw it, from the component or a boundary enclosing it to the other component or a boundary enclosing that one. An edge the view does not draw fails at the reference, naming the view and any drawn edge in the other direction. A drawn edge no reference uses is allowed.

| Language | Reference                                                                                | Line reported                   |
| -------- | ---------------------------------------------------------------------------------------- | ------------------------------- |
| Go       | An import of a folder in this repository, resolved through the nearest tracked `go.mod`  | The import and each `pkg.X` use |
| Swift    | An identifier a top-level type, protocol, type alias, or function in another file declares | The line naming it              |

Comments and string literals create no reference. A symbol two Swift files declare resolves to neither. Member access after a dot is not a reference. A Go file that does not parse fails.

A reference under `//go:build` or `#if` keeps its condition: the error says `under go:build linux` or `under #if DEBUG`. Generated sources stay in the inventory but are not read.

## Evidence

An edge row that links a source file requires the edge; a row without a link allows it. The link must be a Go or Swift source the edge's first component owns. Its lines, or the whole file without an anchor, must name the second component: import its folder, or name a symbol it declares. Anything else fails, naming the row and the link.

For an external system the lines must name it outside comments, by a word of its name, technology, or description.

## Runtimes

The runtimes are the products `architecture-containers` matches to containers. A component under a boundary whose ID is a container of the containers view belongs to that runtime. Each runtime is walked from the components its product folders are built from, along the observed edges.

Reaching a component declared under another runtime's container fails at the reference, naming that boundary. A component outside every container boundary is shared and may be reached by any runtime.

## Scope

Every run reads the whole model and every product source, so a change to only the code or only the view still fails.

## Requirements

[Requirement 0074](../../../docs/requirements/0074-checks-architecture-components.md) has one installed-command test per acceptance.
