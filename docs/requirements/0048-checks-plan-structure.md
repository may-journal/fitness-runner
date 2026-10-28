# 0048 Checks Plan Structure

## Why

I approve a Plan knowing it has a pitch, background, and concrete steps, with no open questions left.

## Measurement

plan bodies that follow the Plan template
-
plan bodies checked

## Requirements

- 0048.1
    - Given Developer has a plan with a pitch, Background, and a checklist step
        - When plan-structure checks the body
            - Then it passes
- 0048.2
    - Given Developer has a plan with no blockquote pitch
        - When plan-structure checks the body
            - Then it fails asking for a one-line pitch
- 0048.3
    - Given Developer has a plan whose pitch still holds `REPLACE-ME`
        - When plan-structure checks the body
            - Then it fails saying the pitch is still the placeholder
- 0048.4
    - Given Developer has a plan without a `## Background` section
        - When plan-structure checks the body
            - Then it fails naming the missing section
- 0048.5
    - Given Developer has a plan whose only checkbox sits inside a code block
        - When plan-structure checks the body
            - Then it fails saying the steps have no checklist item
- 0048.6
    - Given Developer has a plan with an extra `## Notes` section
        - When plan-structure checks the body
            - Then it fails naming the only allowed sections
- 0048.7
    - Given Developer has a plan with a `## Open questions` section
        - When plan-structure checks the body
            - Then it fails asking to turn unknowns into checklist steps
- 0048.8
    - Given Developer has no plan body at all
        - When plan-structure runs
            - Then it passes having judged 0 files
