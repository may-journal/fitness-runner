---
fitnessFunctions: ['architecture-contract', 'architecture-actors']
relatedConfigurations: ['../../.fitnessrc.json']
---

# System Context

```mermaid
C4Context
    title Fitness runner

    Person(developer, "1 Developer")
    Person(agent, "2 AI agent")
    System(fitness, "3 Fitness runner", "Orchestrates repo quality checks for local dev, CI, and GenAI workflows.")
    System_Ext(consumer, "4 Consumer repo", "Application under test")
    System_Ext(tools, "5 Lint format test tools", "eslint, prettier, vitest, cspell, git, etc.")
    System_Ext(ci, "6 CI/CD", "GitHub Actions and similar")

    Rel(developer, fitness, "A")
    Rel(agent, fitness, "B")
    Rel(fitness, consumer, "C")
    Rel(fitness, tools, "D")
    Rel(ci, fitness, "E")
    Rel(fitness, ci, "F")
```

```mermaid
C4Context
    title Legend

    Person(p, "Person", "Human or agent actor")
    System(s, "System", "Software we own")
    System_Ext(e, "External", "Outside our boundary")
```

Numbers on nodes and letters on arrows match the callout table.

| #   | Description                                                                                           | Why                                                                 |
| --- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| 1   | Runs `fitness`, git hooks, or single checks locally.                                          | Primary operator; same bar as CI.                                   |
| 2   | Cursor, Claude Code, or other agents triggered to run fitness before/after edits.                     | GenAI workflows share the runner — see [wardley.md](../wardley.md). |
| 3   | One static Go binary holding the runner and every check, tool configs embedded.                       | The product boundary — a self-contained executable.                 |
| 4   | Any repo with the binaries on PATH; every check runs and skips what does not apply.                   | Runner always executes in the consumer's working directory.         |
| 5   | Commodity linters, formatters, test runners, and git.                                                 | Checks wrap these; runner does not replace them.                    |
| 6   | Pipeline that runs fitness on push or PR.                                                             | Cloud enforcement of the same check list.                           |
| A   | Local dev and optional git hooks.                                                                     | Fast feedback before commit.                                        |
| B   | Agent skills, rules, or hooks invoke the same CLI.                                                    | Closes local vs agent drift when wired.                             |
| C   | Reads staged files, source tree, and config exceptions from the consumer repo.                        | The working directory is the app root, not the runner repo.         |
| D   | Tool-wrapper checks exec the underlying CLIs; native engines need none.                               | Orchestration, not reimplementation.                                |
| E   | CI builds or downloads the binaries and runs `fitness`.                                               | Same command as local.                                              |
| F   | Non-zero exit fails the pipeline.                                                                     | Pass/fail is deterministic, not LLM-judged.                         |
