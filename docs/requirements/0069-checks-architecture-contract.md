---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0069 Architecture Contract

## Why

I keep one clear architecture contract beside my diagrams, so checks can share the same facts.

## Measurement

contracts read or rejected with clear source locations
-
contracts checked

Source: #285

## Requirements

- 0069.1
    - Given no architecture
        - When I check its contract
            - Then it passes with zero files
- 0069.2
    - Given complete diagrams in either root
        - When I renumber callouts
            - Then the contract still passes
- 0069.3
    - Given a partial architecture
        - When I check it
            - Then missing views fail
- 0069.4
    - Given bad declarations
        - When I check them
            - Then errors name their locations
- 0069.5
    - Given unowned or multiply owned sources
        - When I check ownership
            - Then errors name both sides
- 0069.6
    - Given unreadable proof
        - When I check evidence
            - Then errors name its declaration and source
- 0069.7
    - Given invalid requirement links or clauses
        - When I check the contract
            - Then errors name the broken reference
- 0069.8
    - Given unchanged broken evidence
        - When I check unrelated changes
            - Then the same findings remain
