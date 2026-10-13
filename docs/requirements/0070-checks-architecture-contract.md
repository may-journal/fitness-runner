---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0070 Architecture Contract

## Why

I read one model from the diagrams and callout tables I already keep, so checks can share the same facts.

## Measurement

architecture views read or rejected with clear source locations
-
architecture views checked

Source: #285

## Requirements

- 0070.1
    - Given Developer has no architecture
        - When I check its contract
            - Then it passes with zero files
- 0070.2
    - Given Developer has complete diagrams in either root
        - When I renumber their callouts
            - Then the same diagram IDs pass
- 0070.3
    - Given Developer has a partial architecture
        - When I check it
            - Then missing views fail
- 0070.4
    - Given Developer has duplicate, invalid, or unresolved diagram IDs or actor names
        - When I check them
            - Then errors name their diagram lines
- 0070.5
    - Given Developer has two elements naming one source path
        - When I check ownership
            - Then errors name both claims
- 0070.6
    - Given Developer has a source link the check cannot read
        - When I check evidence
            - Then errors name its row and source
- 0070.7
    - Given Developer has invalid requirement links
        - When I check the contract
            - Then errors name only the broken reference
- 0070.9
    - Given Developer has unchanged broken evidence, container drift, or component drift
        - When I check unrelated changes
            - Then the same findings remain
