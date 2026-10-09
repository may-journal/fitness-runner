---
relatedConfigurations: ['../../.fitnessrc.json']
---

# 0043 Checks Mermaid Level Bleed

## Why

I read each lower C4 level for new detail. A callout that only restates its parent level wastes my time.

## Measurement

callout descriptions that differ from the previous level
-
callout descriptions in numbered architecture level files

Source: `go/cmd/fitness-check-mermaid-level-bleed/main.go:158`

## Requirements

- 0043.1
    - Given a level file repeating a callout description from the level above
        - When mermaid-level-bleed runs
            - Then it fails naming the file, the description, and the level
- 0043.2
    - Given adjacent levels with distinct or blank descriptions
        - When mermaid-level-bleed runs
            - Then it passes
- 0043.3
    - Given a repeat that differs only in case or spacing, listed twice
        - When mermaid-level-bleed runs
            - Then it fails, reporting that description once
- 0043.4
    - Given a description shared only by levels 1 and 3
        - When mermaid-level-bleed runs
            - Then it passes, since only adjacent levels are compared
- 0043.5
    - Given level files numbered 2 and 10 sharing a description
        - When mermaid-level-bleed runs
            - Then it fails saying level 10 repeats level 2
- 0043.6
    - Given Markdown outside numbered files directly in an `architecture` folder
        - When mermaid-level-bleed runs
            - Then it passes having judged 0 files
- 0043.7
    - Given callout tables with a `Description` header in any column
        - When mermaid-level-bleed runs
            - Then it compares only that column
- 0043.8
    - Given callout tables without a `Description` header
        - When mermaid-level-bleed runs
            - Then it compares the second column
