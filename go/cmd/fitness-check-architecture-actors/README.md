---
fitnessFunctions: ['architecture-actors']
relatedConfigurations: ['../../../.fitnessrc.json']
---

# architecture-actors

Requires each requirement's Given to start with an actor the system context declares. A role no diagram knows cannot slip into a requirement, and a dropped role fails every Given that still names it.

The check reads the model that [architecture-contract](../fitness-check-architecture-contract/README.md) reads and the Given text that [requirements](../fitness-check-requirements/README.md) parses. It adds no fields, columns, or rows.

## Actors

An actor is a person in `01-system-context.md`: a C4 `Person` or `Person_Ext`. Its name is the label without the callout number, and its Mermaid ID is the one alias the diagram declares.

| Diagram | Name | Alias |
| --- | --- | --- |
| `Person(developer, "1 Developer")` | `Developer` | `developer` |
| `Person_Ext(host, "2 Host")` | `Host` | `host` |

The system context is a `C4Context` diagram, as its [template](../../internal/architecture/templates/01-system-context.md) draws it. A flowchart node classed `person` is no actor, so a Given that names one fails until the view becomes C4. Systems, legends, and people in other views are never actors.

## Grammar

A Given starts with one actor, or several joined by `and`, then the state. A spelling matches only as a whole word, with the diagram's case.

| Given | Result |
| --- | --- |
| `Given Host has no network` | Passes |
| `Given host has no network` | Passes, by ID |
| `Given Host and Participant have joined` | Passes |
| `Given Hosts have joined` | Fails: no whole-word actor |
| `Given Admin has joined` | Fails: undeclared |
| `Given I have joined` | Fails unless `I` is an actor's name or ID |
| `Given no network` | Fails: no actor |
| `Given Host and` | Fails: no actor after `and` |

Not every actor needs a Given; the check constrains the roles requirements use.

## Failures

- No actor at the start, or after `and`: the Given line, the context diagram line, and the declared names.
- A spelling that is one actor's ID and another actor's name: the Given line and both context lines.

Every run reads every requirement doc and the whole model, scoped runs included, so a removed actor fails even when no requirement changed.

## Behavior

- No architecture views: pass with `filesChecked: 0`.
- Otherwise `filesChecked` counts the requirement docs read.
- Doc shape and missing views belong to the `requirements` and `architecture-contract` checks.

## Requirements

[Requirement 0074](../../../docs/requirements/0074-checks-architecture-actors.md) has one installed-command test per acceptance.
