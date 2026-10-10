---
name: add-requirements
description: Give a project stated requirements, each proven by one test that runs the real product the way a user does. Use when adopting the method in a new project, adding a component to it, or reviewing tests for coverage of stated behavior.
relatedConfigurations: ['../../.fitnessrc.json']
---

# Add requirements proven by real runs

A requirement doc states what a user can do, in the user's voice. Each acceptance in it is owned by exactly one test. That test runs the built product on a real example and asserts only what the acceptance promises. The outside world is replayed from saved answers, so the suite runs offline and fast.

The canonical copy lives in fitness-runner at `skills/add-requirements`. On the shared Mac, `~/agents/skills/add-requirements` links to it, so every agent finds it.

fitness-runner did this for itself: 62 docs under [docs/requirements](../../docs/requirements), one test package under [go/requirements](../../go/requirements). Read [0061-engine-installer.md](../../docs/requirements/0061-engine-installer.md) and [0061_test.go](../../go/requirements/0061_test.go) side by side before starting.

## Write one doc per user-facing area

The doc lives at `docs/requirements/NNNN-<prefix>-<kebab-name>.md`. The [requirements check README](../../go/cmd/fitness-check-requirements/README.md) holds the template and every rule the check enforces. Read it first and link to it, do not restate it.

- Pick a prefix per kind of area. This repo uses `checks-`, `engine-`, `hooks-`, and `github-`; a project chooses its own.
- Why: one or two sentences in the user's voice, saying what holds once the requirement is met.
- Measurement: a numerator line, a `-` line, a denominator line, then `Source:` naming one real code line or one issue.
- Requirements: `- NNNN.N`, then exactly one Given, When, and Then, each nested four spaces deeper.
- Hold a doc to one to eight acceptances. A second Given, When, or Then is a new acceptance.
- Never reuse an ID. Retiring one leaves a gap, and the check reads git history to enforce that.

## Name one test per acceptance

Exactly one test owns each acceptance, and its name carries the ID: in Go, `Test0001_1` owns 0001.1. A test named for an acceptance that no doc defines fails. Delete the test when you delete the acceptance.

The requirements check enforces this naming for Go today. Swift comes next, piloted on May Photos, and this skill gains a Swift section from that pilot. Until then, note the gap for the owner instead of inventing a Swift or TypeScript rule.

## Drive the real product

Each test does what the acceptance's user does: it runs the built binary, app, or CLI on a real example repo or fixture. It asserts only what the Then promises: output, exit code, files written, or calls made. It never calls an internal function, and the product is never mocked.

In this repo, [user_test.go](../../go/requirements/user_test.go) shows the pattern. The `example` helper copies a tracked example repo and commits the Given's files over it. The `fitness` helper runs the documented one-line command, and `sees` asserts the words a user reads plus the exit code.

## Replay the outside world

GitHub, package registries, and external tools are never called live. Save their happy and unhappy answers once, then replay them. Two stand-in patterns cover most cases, both in [replay_test.go](../../go/requirements/replay_test.go):

- A program on PATH: the test binary is symlinked under the program's name, first on PATH.
- Run under that name, the test binary answers from saved responses and logs every call it received.
- A registry or web service: a local HTTP server answers each path from saved bodies and 404s the rest.

Each test owns its saved answers, and asserts on the call log when the Then promises a call was made or not made. [0013_test.go](../../go/requirements/0013_test.go) replays `gh` for Plan verdicts; [0022_test.go](../../go/requirements/0022_test.go) serves a local npm registry.

To meet a machine missing one tool, build a PATH holding every real tool but that one. `pathWithout` in [tools_test.go](../../go/requirements/tools_test.go) does this.

## Prove each test fails for the right reason

A test that cannot fail proves nothing. For every new test, flip its Given, run it, and read the failure. The message must name the promise the Then makes, not a setup error. Restore the Given and run again before moving on.

## Delete what the real runs repeat

Once an acceptance's real run covers a behavior, the unit test that stitched internals together to reach the same behavior is a duplicate. Delete it. Code that no acceptance reaches is either an unstated requirement or dead: add an acceptance, or delete the code.

## Measure the real runs

Coverage of the real runs shows the unstated requirements. In this repo, `make requirements-coverage` builds the product with coverage, runs the suite, and prints coverage by package and in total. Elsewhere in Go, build with `go build -cover` and set `GOCOVERDIR` for the runs. Read the uncovered lines as a list of questions: which promise reaches this, and is it written down?

## Keep the suite fast

Every test calls `t.Parallel()` and builds its own temporary repo, so nothing is shared. One command builds the candidate and runs everything; here that is `make requirements`, which runs [tests/requirements.sh](../../tests/requirements.sh). Tests wait on subprocesses rather than the CPU, so a high parallel count is safe. This repo runs 432 tests in about 30 seconds, 32 at once.

## Fan out across a big codebase

For a large product, split the work by component and run one agent per component.

- Assign the requirement IDs up front, one range per component, so two agents never claim the same number.
- Give each agent its own working copy and branch, following the shared Mac rules in `~/CLAUDE.md`.
- Prefix every helper name with the component, as `dc` and `eslint` prefix helpers here, so merged test files do not collide.
- Each agent writes the docs, the tests, proves each test fails right, and deletes the unit tests its runs repeat.
- The orchestrator collects the branches, runs the whole suite once, and reports the uncovered lines as open questions.

## Adopting in a new project

Convert every component before turning the requirements check on, so the project is never red.

- List the components and assign each a block of requirement IDs.
- Build the shared helpers first: copy the example, run the real product, assert what the user sees, replay what leaves the machine.
- For each component, write its doc from what the product promises today, with one to eight acceptances.
- Write one test per acceptance, named for it, and prove each fails for the right reason.
- Delete the unit tests the real runs repeat, and fan out across components as above.
- Measure coverage and turn each uncovered promise into an acceptance or a deletion.
- With no unit test left unmapped, turn on the requirements check through [docs/adoption.md](../../docs/adoption.md).

## Doc conventions

Fitness judges every tracked markdown file, requirement docs and this skill included. [docs/checks.md](../../docs/checks.md) lists the checks, and each check's README states its limits; the project's own fitness run reports any you miss.
