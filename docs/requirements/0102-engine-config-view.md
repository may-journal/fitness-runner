# 0102 Config View

## Why

I see which checks run in this repo, and which setting made that so, without reading the config or the code.

## Measurement

checks `fitness config` lists with a state and a reason
-
checks the suite knows

## Requirements

- 0102.1
    - Given AI agent has a repo whose config turns off a check and sets one limit
        - When `fitness config --json` runs
            - Then it shows each setting's value and source, and each check enabled or skipped with its reason
- 0102.2
    - Given AI agent has a repo
        - When `fitness config schema` runs
            - Then it prints a JSON Schema giving each key's type, description and default, refusing unknown keys
- 0102.3
    - Given AI agent has a repo whose org policy config turns off a check
        - When `fitness config` runs
            - Then it names the org rule that would stop a run and exits 1
- 0102.4
    - Given Developer has fitness source code that reads config keys through the sections it registers
        - When `fitness config schema` runs
            - Then the schema describes every key each registered section reads
