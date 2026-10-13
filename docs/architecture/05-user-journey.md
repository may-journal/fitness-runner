---
relatedConfigurations: ['../../.fitnessrc.json']
---

# User Journey

## Legend

```mermaid
flowchart TB
    subgraph types["Node types"]
        persona((1 Persona)):::persona
        screen[2 Screen]:::screen
        action(3 Action):::action
        system[(4 System)]:::system
    end

    classDef persona fill:#eef2ff,stroke:#6366f1,color:#312e81,stroke-width:2px
    classDef screen fill:#f8fafc,stroke:#64748b,color:#1e293b,stroke-width:2px
    classDef action fill:#fff7ed,stroke:#f97316,color:#9a3412,stroke-width:2px
    classDef system fill:#ecfdf5,stroke:#10b981,color:#064e3b,stroke-width:2px

    style types fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
```

Numbers match the callout table. "Screen" is a terminal output state, not a GUI view — this is a CLI.

| #   | Description | Why                                                        |
| --- | ----------- | ---------------------------------------------------------- |
| 1   | Persona     | Developer or AI agent — same CLI, same exit-code contract. |
| 2   | Screen      | What the invoker sees printed to the terminal.             |
| 3   | Action      | A command, flag, or file edit the invoker performs.        |
| 4   | System      | Runner-internal step: resolve, load, execute, exit.        |

## Developer/agent journey

Each step's label ends with the requirement docs it fulfils, by ID. The `architecture-journey` check requires every doc in some label and an ID on every step.

```mermaid
flowchart TB
    dev((1 Developer or agent)):::persona
    install(14 Install fitness<br/>Req 0061):::action
    terminal[2 Terminal<br/>Req 0001]:::screen
    runFull(3 Run fitness<br/>Req 0001):::action
    runSingle(4 Run single check<br/>Req 0001):::action
    commitHook(15 Commit through hooks<br/>Req 0009, 0011, 0063):::action
    pushHook(16 Push through hooks<br/>Req 0010, 0066, 0067, 0068):::action
    resolveConfig[(5 Resolve config<br/>Req 0003)]:::system
    passArgs[(6 Passthrough args<br/>Req 0052)]:::system
    executeChecks[(7 Execute checks<br/>Req 0001)]:::system

    subgraph checks["Check families"]
        commitChecks[(17 Commit messages<br/>Req 0020, 0031, 0049, 0056)]:::system
        changelogChecks[(18 Changelog<br/>Req 0017, 0018, 0019, 0054)]:::system
        proseChecks[(19 Prose and templates<br/>Req 0006, 0007, 0021, 0023, 0044, 0058)]:::system
        markdownChecks[(20 Markdown files<br/>Req 0033, 0034, 0035, 0036, 0037)]:::system
        mermaidChecks[(21 Mermaid diagrams<br/>Req 0038, 0039, 0040, 0041, 0042, 0043)]:::system
        architectureChecks[(22 Architecture views<br/>Req 0070, 0072, 0073, 0074, 0075, 0076)]:::system
        requirementChecks[(23 Requirements and tests<br/>Req 0005, 0065, 0071)]:::system
        goSwiftChecks[(24 Go and Swift code<br/>Req 0026, 0027, 0028, 0029, 0030, 0057)]:::system
        nodeChecks[(25 Node packages and tests<br/>Req 0022, 0024, 0045, 0047, 0052, 0059, 0060)]:::system
        repoChecks[(26 Repo hygiene<br/>Req 0016, 0025, 0032, 0046, 0053, 0055)]:::system
    end

    resultsTable[8 Results table<br/>Req 0001]:::screen
    exitZero[(9 Exit 0<br/>Req 0001)]:::system
    failuresShown[10 Failures shown<br/>Req 0004]:::blocked
    fixIssues(11 Fix issues<br/>Req 0004):::action
    configError[(12 Config error<br/>Req 0008)]:::blocked
    ciRun[(13 CI runs fitness<br/>Req 0012, 0013, 0015, 0048)]:::system
    prCheck[(27 Pull request check<br/>Req 0014, 0050, 0051, 0069)]:::system
    release[(28 Release build<br/>Req 0062, 0064)]:::system

    dev --> install
    install --> terminal
    terminal --> runFull
    terminal --> runSingle
    terminal --> commitHook
    terminal --> pushHook
    commitHook --> runFull
    pushHook --> runFull
    runFull --> resolveConfig
    runSingle --> executeChecks
    runSingle --> passArgs
    resolveConfig --> executeChecks
    resolveConfig --> configError
    executeChecks --> checks
    checks --> resultsTable
    resultsTable --> exitZero
    resultsTable --> failuresShown
    failuresShown --> fixIssues
    fixIssues --> runFull
    ciRun --> resolveConfig
    ciRun --> prCheck
    ciRun --> release

    classDef persona fill:#eef2ff,stroke:#6366f1,color:#312e81,stroke-width:2px
    classDef screen fill:#f8fafc,stroke:#64748b,color:#1e293b,stroke-width:2px
    classDef action fill:#fff7ed,stroke:#f97316,color:#9a3412,stroke-width:2px
    classDef system fill:#ecfdf5,stroke:#10b981,color:#064e3b,stroke-width:2px
    classDef blocked fill:#fef2f2,stroke:#ef4444,color:#991b1b,stroke-width:2px

    style checks fill:#ffffff,stroke:#e2e8f0,color:#475569,stroke-width:1px
```

Numbers match the callout table. System context: [01-system-context.md](01-system-context.md). Components: [03-components.md](03-components.md#resolve-check-specs-priority). Code: [04-code.md](04-code.md).

| #   | Description                         | Type    | Why                                                                                   |
| --- | ----------------------------------- | ------- | ------------------------------------------------------------------------------------- |
| 1   | Developer or AI agent.              | Persona | Agent invokes via hook or script; same CLI as a human — no separate agent-only path.  |
| 2   | Terminal / shell.                   | Screen  | Repo root; the working directory is the app under test, not the runner's own source.  |
| 3   | `fitness` (or the pre-commit hook). | Action  | Full-suite run — every session's most common entry point.                             |
| 4   | `fitness <name>`.                   | Action  | Single-check mode bypasses list resolution entirely.                                  |
| 5   | Resolve check list.                 | System  | Every built-in check; checks that do not apply pass clean.                            |
| 6   | Passthrough args forwarded.         | System  | Args after the check spec reach the check unchanged (e.g. `prettier --write .`).      |
| 7   | Run loop executes each check.       | System  | Parallel goroutine pool; every check is a subprocess with its own group-kill timeout. |
| 8   | Results table printed.              | Screen  | Pass/fail per check, totals, ANSI formatting.                                         |
| 9   | Exit 0.                             | System  | All checks passed — deterministic, not LLM-judged.                                    |
| 10  | Failures shown in table.            | Screen  | Exit 1; failing check's `errors` printed inline.                                      |
| 11  | Fix and rerun.                      | Action  | Developer/agent edits source, returns to #3.                                          |
| 12  | Bad config or missing binary.       | System  | Fails loud — every check ships together, so a missing binary is a broken install.     |
| 13  | CI runs the same command.           | System  | Same `fitness` invocation gates the merge, and Plan issues are judged there too.      |
| 14  | Installer puts `fitness` on PATH.   | Action  | One verified download gives every check; nothing else to install.                     |
| 15  | `git commit` runs the hooks.        | Action  | Hooks run the suite, stamp the changelog, and keep each issue linked once.            |
| 16  | `git push` runs the hooks.          | Action  | A push must trace to an approved Plan, with branch metadata checked first.            |
| 17  | Commit message checks.              | System  | Semantic subject, attribution, Plan trailer, and one link per issue.                  |
| 18  | Changelog checks.                   | System  | Each change has one dated entry whose bullets match the change.                       |
| 19  | Prose and template checks.          | System  | Docs stay short, readable, spelled right, and shaped by their templates.              |
| 20  | Markdown file checks.               | System  | File names, front matter, links, and emphasis follow one convention.                  |
| 21  | Mermaid diagram checks.             | System  | Diagrams carry legends, callouts, and tables that explain each number.                |
| 22  | Architecture view checks.           | System  | Views match actors, runtimes, folders, symbols, and this journey.                     |
| 23  | Requirement and test checks.        | System  | Each acceptance has one test, and a doc changes with its test.                        |
| 24  | Go and Swift code checks.           | System  | Format, vet, tests, coverage, complexity, and lint gate the code.                     |
| 25  | Node package and test checks.       | System  | Lint, format, Node version, dependencies, and test coverage stay current.             |
| 26  | Repo hygiene checks.                | System  | No build output, plan files, unexplained ignores, or copied code.                     |
| 27  | Pull request check.                 | System  | A pull request's body, closed issues, and changelog link are judged in CI.            |
| 28  | Release build.                      | System  | The release tool builds bundles and publishes notes that link the changelog.          |
