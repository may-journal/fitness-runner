---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0010 — One binary holds every check

Check: [fitness/adr-0010](../../../fitness/adr-0010/README.md)

## Context

Each platform shipped 63 binaries, 219 MB unpacked, which linked the same packages 63 times. Releases spent most of their build linking, and installs most of their time downloading.

## Decision

1. Every `go/cmd/fitness-check-<name>` folder is a package that exports `Main`, built into `fitness`.
2. `fitness` runs a check when started as `fitness-check-<name>`, or as `fitness check <name>`.
3. `fitness-release checks` writes the name list from the folders, so no one keeps it by hand.
4. The runner still starts each check as its own process, under the check's own name.
5. A `fitness-check-<name>` program beside the runner replaces the check of that name, as before, and the runner says so on each run.
6. The installer links `fitness-check-<name>` to `fitness` for every check, so scripts that call a check by name keep working.

## Consequences

1. A bundle is one 14 MB binary in a 6 MB archive, down from 219 MB and 97 MB.
2. The bundle build drops from 19 s to 3 s here, and the release links 4 binaries, not 252.
3. Checks still cannot share state, since each one is a process; every check process loads every check's code.
4. A new check folder needs `go generate` before the binary knows it; until then its requirement tests fail with an unknown check.
