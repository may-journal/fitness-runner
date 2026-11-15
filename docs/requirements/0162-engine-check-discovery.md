# 0162 Engine check discovery

## Why

Each check can ship as its own program. So the engine finds installed checks, agrees a protocol version with each, and names the install command for a check it lacks.

## Measurement

Installed check programs the engine finds, version-checks and runs as documented
-
Installed check programs

## Requirements

- 0162.1
    - Given Developer has check programs in the `FITNESS_CHECKS_DIR` folder, beside fitness and on `PATH`
        - When I run them
            - Then each name runs from the first of those places that holds it, and the built-in `local-branch` reports protocol 1 to `--describe`
- 0162.2
    - Given Developer has an installed check whose `--describe` reports protocol 2, and one that reports none
        - When I run each
            - Then the first stops the run, naming the check, both versions and its install command, and the second sees `FITNESS_PROTOCOL=1`
- 0162.3
    - Given Developer has one installed check
        - When I run the suite with `--no-builtins`, or name with it a built-in check not installed, also through `fitness check`
            - Then the suite runs only the installed check, and the selection fails naming the check and its install command
- 0162.4
    - Given Developer has an installed program for `local-branch`, whose judgment lives in the engine
        - When I run it, with or without `--no-builtins`
            - Then fitness says the program only declares the check, judges itself and never runs it, in place of 0162.1 and 0162.3
- 0162.5
    - Given Developer has a Go test that runs fitness, in a repo that `go-test` judges
        - When `go-test`, built on this engine, runs that test
            - Then the test's fitness finds the installed checks, in place of printing no result
