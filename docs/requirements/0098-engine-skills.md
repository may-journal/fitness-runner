---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0098 Skills reach every agent

## Why

Every skill in a Fitness release lands in each agent's skills folder when I run one command, so no agent misses a skill. Skills I made myself stay as they are.

## Measurement

Agent skills folders that hold every release skill after linking
-
Agent skills folders linked

Source: `go/internal/distribution/skills.go:20`

## Requirements

- 0098.1
    - Given Developer has an empty skills folder for an agent
        - When I run `fitness-install --link-skills` with that folder
            - Then every skill in the release is linked there and readable
- 0098.2
    - Given Developer has a broken skill link and a skill folder of my own in an agent's skills folder
        - When I run `fitness-install --link-skills` with that folder
            - Then the broken link is repaired and my own folder stays untouched
- 0098.3
    - Given Developer asks for the commit hook and skill links in one command
        - When I run `fitness-install --install-hook --link-skills` with a folder
            - Then it refuses with an error and links nothing
- 0098.4
    - Given Developer has a link to a dropped skill and a link into another tool's bundle in a skills folder
        - When I run `fitness-install --link-skills` with that folder
            - Then the link to the dropped skill is removed and the other tool's link stays
- 0098.5
    - Given Developer has an empty skills folder for an agent
        - When I run `fitness-install --link-skills` with that folder
            - Then every linked skill file is readable by other users
