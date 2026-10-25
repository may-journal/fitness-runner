---
name: requirements
description: Fix a failing requirements check on requirement docs and their tests. Use when a requirement has no test, a test has no requirement, or a doc and its test changed apart.
---

# Pair requirements with tests

Each file in `docs/requirements` states requirements, and each acceptance is owned by exactly one test named for it. [The check README](../../go/cmd/fitness-check-requirements/README.md) holds the doc template and every rule. To add requirements to a project, follow the [add-requirements skill](../add-requirements/SKILL.md).

## Read the failure

- `<doc>:<line>: 0001.1 has no test; name exactly one test <name>`: write the test with that name, one that runs the real product.
- `<doc>:<line>: 0001.1 is owned by 2 tests (a, b); keep exactly one`: delete the duplicate.
- `<location>: <test> names 0001.9, which no requirement defines; delete the test with its acceptance`: delete the test, or add the acceptance to its doc.
- `<location>: <test> proves no requirement; rename it <name> for the acceptance it proves, or delete it if another test covers that`: rename it or delete it.
- `<file> changed without its requirement doc <path>; change both together`: change the test and its doc in the same commit. A changed doc names its requirement test the same way.

## Check it first

Run `fitness requirements` from the repo root. Exit 0 means every test and doc pair up.
