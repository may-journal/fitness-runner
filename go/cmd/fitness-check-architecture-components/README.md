---
fitnessFunctions: ['architecture-components']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-components

Gives every product source one owner among the components the components view draws. It keeps the references between those components, and the runtimes that reach them, to what the view draws. It reads the model `architecture-contract` describes and adds no fields or columns.

## Owners

A component owns the product sources its claims cover: a path in its diagram strings or its callout row. Boundaries, people, and external systems own nothing; their claims only scope the components inside them.

Product sources are the product category of the `architecture-contract` inventory, so tests, tools, and generated files need no owner. These sources are tools:

- A root-level tool config such as `rollup.config.mjs` or `.eslintrc.cjs`; `src/app/app.config.ts` is product code.
- A source under the top-level `scripts/` folder.
- A file only `package.json` scripts run, as in `node tools/gen.mjs`. A file the `start` script runs or the package publishes stays product code.

Ownership fails or holds as follows:

- A product source no component names fails, naming the source.
- A code span or label claims a root-level source by its bare name, such as `index.ts`; `package.json` stays prose.
- A source two components name, in two diagrams of one view, fails naming both claims. Two claims in one diagram fail in `architecture-contract`.
- A claimed path that no longer exists fails in `architecture-contract`.

## References

Every reference between two owners is an observed edge. The view must draw it, from the component or a boundary enclosing it to the other component or a boundary enclosing that one. An edge the view does not draw fails at the reference, naming the view and any drawn edge in the other direction. A drawn edge no reference uses is allowed.

| Language | Reference                                                                               | Line reported                   |
| -------- | --------------------------------------------------------------------------------------- | ------------------------------- |
| Go       | An import of a folder in this repository, resolved through the nearest tracked `go.mod` | The import and each `pkg.X` use |
| Swift    | A name the Swift parser resolves to a declaration in another file                       | The line naming it              |

Comments and string literals create no reference; a name inside a string interpolation does. A Go file that does not parse fails.

## Swift names

[ADR 0005](../../../docs/architecture/adr/0005-swift-references-from-a-go-parser.md) explains the Swift parser. It reads every product Swift file as one module and resolves a name the way the compiler would:

- A local name, such as a parameter or a binding in a body, hides any declaration and reaches nothing.
- In a type or extension body, that type's members, from any file, come before file-level declarations.
- A member after a dot is looked up on its base's type, with members from extensions and inherited protocols.
- The base's type is known for a type name, `self`, a literal, or a value whose type is stated or initialized.
- A base of unknown type, or an implicit member such as `.loaded`, reaches nothing.
- A call keeps the overloads its argument labels fit; candidates left in two files reach none.
- A call of a type also reaches the initializer its labels pick.
- In an `#if` group whose branches leave a brace open, as alternative headers do, only the first branch is read.

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

[Requirement 0075](../../../docs/requirements/0075-checks-architecture-components.md) has one installed-command test per acceptance.
