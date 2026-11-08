# 0149 ADR Migrate

## Why

One command moves my ADR files into `ADR` issues, so no decision is lost or opened twice.

## Measurement

ADR files moved into one issue each
-
ADR files migrated

## Requirements

- 0149.1
    - Given Developer has ADR files and asks for a dry run
        - When fitness runs adr-migrate with `--dry-run`
            - Then it prints each issue it would open and each file it would change, and changes nothing
- 0149.2
    - Given Developer has ADR files in an `adr` folder and a check README holding its decision
        - When fitness runs adr-migrate
            - Then it opens one `[ADR]` issue per file, points links at it, trims or removes the file, and closes a superseded one
- 0149.3
    - Given Developer reruns adr-migrate after issues for its files exist
        - When fitness runs adr-migrate
            - Then it keeps those issues, points their links at the new issues, leaves a closed issue closed, and opens no duplicate
