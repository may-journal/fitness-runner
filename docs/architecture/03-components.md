---
fitnessFunctions: ['architecture-contract', 'architecture-components']
relatedConfigurations: ['../../.fitnessrc.json']
---

# Components

```mermaid
C4Component
    title Runner CLI

    Person(developer, "1 Developer")

    System_Boundary(cli, "Runner binary (go/cmd/fitness)") {
        Container_Boundary(entry, "Entry") {
            Component(main, "2 CLI entry", "Go")
        }
        Container_Boundary(runner, "Run loop") {
            Component(config, "3 Config loader", ".fitnessrc.json")
            Component(resolve, "4 Check resolver", "allChecks from the module registry")
            Component(execute, "5 Execution pool", "goroutines + group kill")
            Component(output, "6 Results renderer", "internal/render")
        }
    }

    System_Boundary(checks, "Check binaries") {
        Component(check, "7 Check binary", "fitness-check-<name>")
    }

    System_Boundary(shared, "Shared internals") {
        Component(kit, "8 checkkit protocol", "go/internal")
    }

    System_Boundary(tools, "Install and release tools") {
        Component(tool, "21 Tool binary", "Go", "go/cmd/fitness-install, go/cmd/fitness-release")
    }

    System_Ext(peers, "9 Peer tools", "each language module's formatters, linters and test runners")
    System_Ext(git, "10 git")

    Rel(developer, main, "11")
    Rel(main, config, "12")
    Rel(main, resolve, "13")
    Rel(resolve, check, "14")
    Rel(main, git, "17")
    Rel(main, execute, "18")
    Rel(execute, check, "19")
    Rel(main, output, "20")
    Rel(check, kit, "15")
    Rel(check, peers, "16")
    Rel(main, kit, "22")
    Rel(tool, kit, "23")
```

```mermaid
C4Component
    title Legend

    Person(p, "Person", "Human or agent actor")
    System_Boundary(b, "System boundary") {
        Container_Boundary(cb, "Container boundary") {
            Component(c, "Component", "Tech", "Logical unit")
        }
    }
    System_Ext(e, "External", "Outside our boundary")
```

Numbers on nodes and arrows match the callout table.

| #   | Description                                                                                             | Why                                                         |
| --- | ------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| 1   | Developer or agent invokes `fitness`.                                                                   | Same entry as system context.                               |
| 2   | `go/cmd/fitness` — `main.go` routes the subcommands, else argv, resolve, execute, render.               | No build step, no interpreter startup.                      |
| 3   | Loads `.fitnessrc.json` (stdlib encoding/json); a lone legacy script config earns a migration hint.     | Config a compiled runner can parse anywhere.                |
| 4   | Ordered check list: the CLI name, else `allChecks`, which the language module registry lists.           | One place for check resolution (single name or full list).  |
| 5   | Bounded goroutine pool with per-check timeouts; expiry kills the check's whole process group.           | Parallelism bounds wall-clock at the slowest check.         |
| 6   | Renders the results table + totals line from collected outcomes, in dispatch order.                     | User-visible pass, fail and info summary.                   |
| 7   | One static binary per check name (`go/cmd/fitness-check-*`), discovered beside the runner then on PATH. | Plugin model at the artifact level; no registry.            |
| 8   | `checkkit` defines the protocol types and main scaffolding; walker, git, markdown, render helpers.      | One implementation shared by every check binary.            |
| 9   | Tool-wrapper checks exec the real peer tool, found the way its language module finds tools.             | Orchestration, not reimplementation.                        |
| 10  | `git diff --cached --name-only` for staged file context.                                                | Pre-commit and partial runs.                                |
| 11  | User-facing invocation.                                                                                 | Sole human entry point; all else is internal wiring.        |
| 12  | Entry loads config before argv resolution.                                                              | `.fitnessrc.json` drives order, subset, and options.        |
| 13  | Entry delegates to the resolver first.                                                                  | Resolve before execute.                                     |
| 14  | Resolver maps a name to the `fitness-check-<name>` binary and collects its `--describe` metadata.       | Timeout and context-inline metadata without header parsing. |
| 15  | Check binaries build on checkkit for protocol emit and env context.                                     | Contract every check satisfies.                             |
| 16  | Tool wrappers exec their peer binary and parse its output.                                              | The check owns tool invocation and judgment.                |
| 17  | Runner exports `FITNESS_STAGED_FILES` before dispatch.                                                  | Checks receive consistent context.                          |
| 18  | Run loop dispatches each check to the pool.                                                             | Ordered dispatch; buffered, ordered output.                 |
| 19  | Pool execs the binary with `--root`, env context, stdout JSON captured.                                 | One execution model for every check.                        |
| 20  | Run loop prints the table after all checks finish.                                                      | Single summary per invocation.                              |
| 21  | The installer and the release tool, one static binary each.                                             | Shipped beside the checks; not part of any run.             |
| 22  | Entry builds on checkkit, conf, selection, render, and report.                                          | The runner shares the protocol it asks checks to speak.     |
| 23  | Tools build on the release, distribution, and changelog packages.                                       | One implementation of versions and bundles.                 |

## Resolve check specs (priority)

`go/cmd/fitness` produces an ordered name list, then resolves each name to `fitness-check-<name>` beside the runner, then on PATH:

1. A CLI name (`fitness prettier`) — run only that built-in check; an unknown name is an error.
2. Otherwise — every installed check minus [ignores](../../go/internal/lang/ignores): unclaimed ones sorted, then each [language module](../../go/internal/lang/langs/langs.go)'s, sorted.

Every check runs in every repo and passes clean with zero files when it does not apply. A missing binary for a module's check fails the run. A leftover `checks` key in `.fitnessrc.json` is ignored with a one-line stderr warning. Each language module reads its own check options from that file, so the shared loader names no language.

## Module data the repo-wide checks read

Repo-wide checks take what a language brings from the registry, never from a list of their own. The spell and duplicate checks skip each module's lockfiles, plus those in one [lockfiles](../../go/internal/lang/lockfiles) file per other ecosystem. The build output check fails tracked files under the output folders each module finds in the repo.

## Coverage goal

Every coverage check judges exact counts against one goal in [goal](../../go/internal/goal/goal.go), whichever module runs it. A module that spots test frameworks keeps one commented file per framework in its own folder, so adding one needs no code.

## Architecture view readers

The architecture checks read code through one model in [codemodel](../../go/internal/codemodel/model.go): declarations, members, references and relations. Each language module brings a reader for it, and the views take every reader from the registry. Readers that read tokens relate names through one shared walk, as [ADR 0008](adr/0008-languages-plug-in-as-modules.md) decides. A language with no module lists its endings in a [sources](../../go/internal/lang/sources) file, so the views still inventory it.

Single-check mode bypasses the list. Passthrough args after the spec reach the check. A check whose `--describe` declares a context-inline argument has that value extracted into the environment and stripped from passthrough. The commit checks declare `--message`.

## Run context (environment)

| Variable                 | Source                                      |
| ------------------------ | ------------------------------------------- |
| `FITNESS_CHECK_NAME`     | resolved name of the running check          |
| `FITNESS_ENABLED_CHECKS` | names in this run, newline-separated        |
| `FITNESS_STAGED_FILES`   | git staged paths (node_modules stripped)    |
| `FITNESS_CTX_MESSAGE`    | `--message` value for context-inline checks |

Built by the runner before dispatch — not a separate registry.

## Body mode

Check binaries also accept a document instead of a file tree. Given `--body-file`, `bodycheck.RunDoc` runs a check's per-document rule on that body, with config from `--root`. The `fitness issue-check`, `plan-check`, `pr-check`, `close-check`, and `comment-check` subcommands use this to judge GitHub blobs: Issue and PR bodies and comments.

An explicit `--body-file` is the only trigger, so a normal file-walking run is never affected. The prose, markdown, and mermaid checks opt in; a cross-file check like `mermaid-level-bleed` does not.

## Subcommands

Before the run loop, the CLI entry routes a few subcommands. Each one lets a git hook or workflow step shrink to one line, while the logic stays in tested Go.

| Subcommand            | Does                                                                                  | Why                                                   |
| --------------------- | ------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| `fitness init`        | Writes the embedded hook shims to `.githooks` and sets `core.hooksPath`.              | A repo pulls its hooks by version, never copies them. |
| `fitness hook <name>` | Runs the commit-msg, pre-commit, or pre-push logic the shims call.                    | Hook logic lives in one place, not in each repo.      |
| `fitness pr-check`    | Judges a PR's title, body, one linked changelog entry, and the issues it closes.   | The PR check workflows run one step.                  |
| `fitness issue-check` | Judges an issue by its kind label: its template's shape and the kind's checks; its after-effect comments and labels the verdict. | The issue check workflows run one step. |
| `fitness plan-check`  | `issue-check` for Plan issues only.                                                  | Repos on plan-check-reusable keep working.            |
| `fitness close-check` | Judges a closed issue with `issue-checklist`; its after-effect reopens it once.       | The close check workflows run one step.               |

`go/internal/metadata/github.go` declares each workflow subcommand as its checks and an after-effect from `go/internal/aftereffect`. For `issue-check`, `go/internal/metadata/kinds.go` reads each kind from `.github/ISSUE_TEMPLATE` and adds the rules a kind label names. The local-branch check shares their judgments. Only after-effects write to GitHub, through the `gh` CLI.
