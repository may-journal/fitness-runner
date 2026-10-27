# 0012 — A repo's own checks live in its fitness folder

Fitness keeps its own checks in its `fitness/` folder.

## Context

[ADR 0009](../adr-0009/README.md) let a repo add flat check programs, with no README, SDK or listing. An ADR stayed prose that no check kept.

## Decision

1. Each `fitness/<name>/` folder is one check, found with no config. (KISS)
2. A check folder holds a README.md and Go source on the `go/check` SDK, built once into the cache. Any executable is the fallback.
3. The SDK gives a check typed inputs and findings, and `check.Repo` and `check.Judge` unit-test it.
4. `fitness check <name>` runs one check alone, and `fitness list` and `fitness explain` show it from its README.
5. In a repo with a `fitness/` folder, each ADR is the README of its `fitness/adr-NNNN/` check, and lives nowhere else.
6. A `fitness/req-NNNN/` check may prove a requirement doc in place of tests. A paired check that judges no files fails.
7. A changed decision needs its check's code to change too, and a superseded ADR needs no code.
8. ADR 0009's rules on replacing and disabling bundled checks stand, and `localChecks` still names one more folder.

## Consequences

1. A repo's rules sit in one folder, each documented and runnable alone like a unit test.
2. A decision holds only while its check passes, and one document holds both, so an ADR no longer drifts from the code.
3. A Go check needs Go on the machine. CI installs it when `fitness/go.mod` exists.
4. Adding a `fitness/` folder moves every ADR into a `fitness/adr-NNNN/` README. Without one, adr-checks passes with no files.
5. A local check is repo code that runs with the developer's or CI's rights, as git hooks do.

## Check

- Fails `.fitnessrc.json` when it names a `localChecks` folder.
- Fails a tracked `fitness-check-<name>` program outside `go/cmd`.
- Fix: move the check into `fitness/<name>/` with a README.md.
