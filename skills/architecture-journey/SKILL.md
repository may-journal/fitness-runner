---
name: architecture-journey
description: Fix a failing architecture-journey check on the user journey view. Use when a requirement doc is missing from the journey or a step names no requirement.
---

# Cover each requirement in the journey

Every requirement doc appears by its four-digit ID on a step of the [journey diagram](../../docs/architecture/05-user-journey.md), and every behavior step names an ID. [The check README](../../go/cmd/fitness-check-architecture-journey/README.md) states the rules.

## Read the failure

- `requirement 0042 is missing from the journey diagram; name it in the label of the step that fulfils it`: add `Req 0042` after the callout number in that step's label.
- `journey node open names no requirement; add the IDs it fulfils to its label or type it as a persona`: add the IDs, or give a person the persona class.
- `0042.1 is an acceptance ID; name requirement doc 0042`: use the four-digit doc ID.
- `no requirement doc 0099 in docs/requirements`: fix the number or add the doc.
- `label names 9 requirements; split the step so each names at most 8`: split the step.

## Check it first

Run `fitness architecture-journey` from the repo root. Exit 0 means the diagram and the requirement docs cover each other.
