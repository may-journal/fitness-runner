# 0114 Generated Check Docs

## Why

A check's README, the check index and the command list say what the installed binary says, because `go generate` writes each from the code.

## Measurement

check READMEs that match `fitness explain`

-

checks in the catalog

Source: `go/internal/release/readmes.go:28`

## Requirements

- 0114.1
  - Given AI agent reads a check's README in the fitness repo
    - When it runs `fitness explain --json` for every check `fitness list` names
      - Then the README holds the same documentation as the explanation
- 0114.2
  - Given AI agent reads the check index in docs/checks.md
    - When it runs `fitness list --json`
      - Then the index links every listed check's README with the same summary
- 0114.3
  - Given Developer reads the command list in the README
    - When they run `fitness --help`
      - Then the help prints the same list
