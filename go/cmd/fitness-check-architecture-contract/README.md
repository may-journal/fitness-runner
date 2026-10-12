---
fitnessFunctions: ['architecture-contract']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-contract

Reads canonical facts from each diagram's callout table. Checks share the parsed model through `internal/architecture.Load` and requirement clauses through `internal/requirement.Load`.

## Views

The roots are `architecture/` and `docs/architecture/`. Both roots contribute to one model. IDs must be unique across both.

| File | View |
| --- | --- |
| `01-system-context.md` | Actors and systems |
| `02-containers.md` | Runtime products |
| `03-components.md` | Components and source ownership |
| `04-code.md` | Files and symbols |
| `05-user-journey.md` | Behaviors and requirement links |

No views means a pass with zero files. Any view requires all five. Scoped runs read the whole model and its tracked source files.

## Contract cells

Add a `Contract` column to the existing numbered table. Each cell holds `key=value` fields, separated by semicolons. Lists use commas.

| Field | Meaning |
| --- | --- |
| `id` | Stable semantic ID, beginning with a letter; letters, digits, underscores, and hyphens follow |
| `ref` | Reuse a node or relationship ID, including a declaration in a later view |
| `type` | `actor`, `system`, `external`, `runtime`, `component`, `symbol`, `behavior`, or `relationship` |
| `parent` | Parent ID; required for runtime, component, and symbol declarations |
| `name`, `aliases` | Actor name and unique alternative names |
| `path`, `symbol` | Tracked source file and symbol name; both required for symbols |
| `owns` | Source files, shell-style path patterns, or directory prefixes ending in `/` |
| `target` | `go:folder`, `swift-package:Package.swift:name`, or `xcode-project:project.yml:name` |
| `requirements` | Requirement document IDs or acceptance IDs |
| `from`, `to` | Directed relationship endpoints, using stable IDs |
| `kind` | `uses`, `calls`, `imports`, `inherits`, `implements`, `contains`, `reads`, `writes`, `sends`, or `navigates` |
| `mode` | `allowed` or `required` relationship constraint |
| `evidence` | Proof reference, described below |

Unknown, empty, repeated, or malformed fields fail at the table row. Unknown IDs, parent cycles, duplicate declarations, and ambiguous actor aliases fail too.

Each node or edge has one numbered callout. Its row binds that display number to a semantic ID. Renumbering both preserves identity.

## Mermaid forms

C4 context, container, and component diagrams support Person, System, Container, and Component macros, including external, database, and queue variants. Boundaries group nodes.

Directed `Rel` and its `_U`, `_D`, `_L`, and `_R` forms bind to relationship rows. Flowcharts and classes support separate node declarations. Arrowheads set semantic direction, including class inheritance arrows that point left.

An explicit `%% architecture:legend` line marks a legend fence. Its examples, including numbered examples, do not enter the model. Existing form checks still apply.

## Source inventory

Every tracked source is retained with an owner and category. Overlapping claims and unowned files fail; generated files remain in the inventory.

| Category | Source |
| --- | --- |
| product | Supported source extensions, unless classified below |
| test | `_test.go` files and complete `test` or `tests` folder names, without regard to case |
| tool | Shell and Python files outside test folders |
| generated | Files marked `linguist-generated` by git attributes |

Go main packages and literal SwiftPM target declarations enter the target inventory. Swift comments and strings do not declare targets. XcodeGen accepts target names under a top-level `targets` key, indented two spaces.

Target IDs resolve only in the build-target inventory. Later checks resolve language-specific product and dependency facts.

## Evidence

| Form | Meaning |
| --- | --- |
| `declared` | An unproved diagram constraint; never treated as source or runtime proof |
| `static:path:line` | An existing line in a tracked source; later checks resolve its relationship or symbol meaning |
| `runtime:path:line` | Reserved runtime trace reference; fails until a child check supplies a trace resolver |

Missing files and lines fail at the declaration and name the requested evidence location. Unknown proof kinds fail. Runtime behavior never passes on a diagram claim alone.

This foundation validates the contract, references, and ownership. Actor equality, deployable products, component imports, symbol relationships, and journey coverage belong to the later checks.

## Requirements

[Requirement 0070](../../../docs/requirements/0070-checks-architecture-contract.md) has one installed-command test per acceptance, using committed Go and Swift fixtures.
