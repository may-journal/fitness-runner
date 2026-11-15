# 0114 Generated Check Docs

## Why

The command list says what the installed binary says, because `go generate` writes it from the code. Each check's README lives in fitness-checks, which generates it.

## Measurement

command list lines that match `fitness --help`

-

command list lines in the README

## Requirements

- 0114.2
  - Given AI agent looks for a check's README in the fitness repo
    - When it lists the check folders under go/cmd
      - Then none holds a README, and docs/checks.md links fitness-checks, which generates each one
- 0114.3
  - Given Developer reads the command list in the README
    - When they run `fitness --help`
      - Then the help prints the same list, its run forms and commands
