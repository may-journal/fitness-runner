---
relatedConfigurations: ['../.fitnessrc.json']
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

```mermaid
flowchart TB
    dev((1 Developer or agent)):::persona
    terminal[2 Terminal]:::screen
    runFull(3 Run fitness):::action
    runSingle(4 Run single check):::action
    resolveConfig[(5 Resolve config)]:::system
    passArgs[(6 Passthrough args)]:::system
    executeChecks[(7 Execute checks)]:::system
    resultsTable[8 Results table]:::screen
    exitZero[(9 Exit 0)]:::system
    failuresShown[10 Failures shown]:::blocked
    fixIssues(11 Fix issues):::action
    authorLocal(12 Author local check executable):::action
    editRc(13 Add path to .fitnessrc):::action
    configError[(14 Config error)]:::blocked
    ciRun[(15 CI runs fitness)]:::system

    dev --> terminal
    dev --> authorLocal
    terminal --> runFull
    terminal --> runSingle
    runFull --> resolveConfig
    runSingle --> executeChecks
    runSingle --> passArgs
    resolveConfig --> executeChecks
    resolveConfig --> configError
    executeChecks --> resultsTable
    resultsTable --> exitZero
    resultsTable --> failuresShown
    failuresShown --> fixIssues
    fixIssues --> runFull
    authorLocal --> editRc
    editRc --> resolveConfig
    ciRun --> resolveConfig

    classDef persona fill:#eef2ff,stroke:#6366f1,color:#312e81,stroke-width:2px
    classDef screen fill:#f8fafc,stroke:#64748b,color:#1e293b,stroke-width:2px
    classDef action fill:#fff7ed,stroke:#f97316,color:#9a3412,stroke-width:2px
    classDef system fill:#ecfdf5,stroke:#10b981,color:#064e3b,stroke-width:2px
    classDef blocked fill:#fef2f2,stroke:#ef4444,color:#991b1b,stroke-width:2px
```

Numbers match the callout table. System context: [01-system-context.md](01-system-context.md). Components: [03-components.md](03-components.md#resolve-check-specs-priority). Code: [04-code.md](04-code.md).

| #   | Description                      | Type    | Why                                                                                                                  |
| --- | -------------------------------- | ------- | -------------------------------------------------------------------------------------------------------------------- |
| 1   | Developer or AI agent.           | Persona | Agent invokes via hook or script; same CLI as a human — no separate agent-only path.                                 |
| 2   | Terminal / shell.                | Screen  | Repo root; the working directory is the app under test, not the runner's own source.                                 |
| 3   | `fitness` or `npm run fitness`.  | Action  | Full-suite run — every session's most common entry point.                                                            |
| 4   | `fitness <name>` / `--check=…`.  | Action  | Single-check mode bypasses list resolution entirely.                                                                 |
| 5   | Resolve check specs.             | System  | `.fitnessrc.json` `checks` (names and/or paths) else the embedded default list; `disabledChecks` filters names only. |
| 6   | Passthrough args forwarded.      | System  | Args after the check spec reach the check unchanged (e.g. `prettier --write .`).                                     |
| 7   | Run loop executes each check.    | System  | Parallel goroutine pool; every check is a subprocess with its own group-kill timeout.                                |
| 8   | Results table printed.           | Screen  | Pass/fail per check, totals, ANSI formatting.                                                                        |
| 9   | Exit 0.                          | System  | All checks passed — deterministic, not LLM-judged.                                                                   |
| 10  | Failures shown in table.         | Screen  | Exit 1; failing check's `errors` printed inline.                                                                     |
| 11  | Fix and rerun.                   | Action  | Developer/agent edits source, returns to #3.                                                                         |
| 12  | Author a local check executable. | Action  | Consumer-repo executable speaking the JSON protocol — a shell script works.                                          |
| 13  | Add its path to `checks`.        | Action  | Path specs (entries containing a separator) opt in explicitly; `disabledChecks` cannot remove them.                  |
| 14  | Missing or invalid path throws.  | System  | Path entries fail loud — explicitly configured, unlike name specs which allow-miss.                                  |
| 15  | CI runs the same command.        | System  | Same `fitness` invocation gates the merge — no CI-only config branch.                                                |
