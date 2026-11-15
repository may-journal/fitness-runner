# 0114 Generated Check Docs

## Why

The check index and the command list say what the installed binary says, because `go generate` writes each from the code. Each check's README lives in fitness-checks, which generates it.

## Measurement

index rows that match `fitness list`

-

checks the engine runs itself

## Requirements

- 0114.1
  - Given AI agent reads the check index in docs/checks.md
    - When it runs `fitness check --list` and `fitness list --json`
      - Then the index names each check the engine runs itself, with the same summary, linking its doc.go
- 0114.2
  - Given AI agent looks for a check's README in the fitness repo
    - When it lists the check folders under go/cmd
      - Then none holds a README, and docs/checks.md links fitness-checks, which generates each one
- 0114.3
  - Given Developer reads the command list in the README
    - When they run `fitness --help`
      - Then the help prints the same list, its run forms and commands
