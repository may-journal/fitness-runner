---
relatedConfigurations: ['../package.json']
---

# Containers

```mermaid
C4Container
    title Fitness runner

    Person(developer, "1 Developer")
    Person(agent, "2 AI agent")

    System_Boundary(publish, "Published packages") {
        Container(cli, "3 Runner CLI", "Node", "@mayjournal/fitness")
        Container(bundle, "4 Checks bundle", "Node", "defaultChecks + meta-package")
        Container(checks, "5 Check packages", "Node", "@mayjournal/fitness-checks/checks/*")
        Container(shared, "6 Shared configs", "Node", "@mayjournal/fitness-shared")
    }

    System_Boundary(consumer, "Consumer repo") {
        Container(rc, "7 Config", ".fitnessrc optional")
        Container(localChecks, "8 Local check modules", "Optional paths in checks")
        Container(code, "9 Application code", "Sources under test")
    }

    System_Ext(npm, "10 npm")
    System_Ext(tools, "11 Tooling", "eslint, prettier, vitest, cspell, git")

    Rel(developer, cli, "12")
    Rel(agent, cli, "13")
    Rel(cli, rc, "14")
    Rel(cli, checks, "15")
    Rel(cli, localChecks, "16")
    Rel(cli, bundle, "17")
    Rel(checks, shared, "18")
    Rel(checks, tools, "19")
    Rel(checks, code, "20")
    Rel(localChecks, code, "21")
    Rel(bundle, checks, "22")
    Rel(cli, npm, "23")
    Rel(rc, shared, "24")
```

```mermaid
C4Container
    title Legend

    Person(p, "Person", "Human or agent actor")
    System_Boundary(b, "System boundary") {
        Container(c, "Container", "Tech", "Runnable or deployable unit")
    }
    System_Ext(e, "External", "Outside our boundary")
```

Numbers on nodes and arrows match the callout table.

| #   | Description                                                                                                                                                                | Why                                                                              |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| 1   | Same actor as system context.                                                                                                                                              | All runs start at the CLI.                                                       |
| 2   | Same agent actor as system context.                                                                                                                                        | Agents never load checks directly — they invoke the CLI.                         |
| 3   | Entry `fitness`, resolve check specs, dynamic import, run loop, results table.                                                                                             | Single orchestration surface.                                                    |
| 4   | `@mayjournal/fitness-checks` — depends on all checks; exports `defaultChecks`.                                                                                             | Default path when `.fitnessrc` omits `checks`.                                   |
| 5   | One workspace per check under `packages/checks/<name>/`; loaded via `@mayjournal/fitness-checks/checks/<name>` (the per-check package itself is private, never published). | Plugin model; no in-process registry.                                            |
| 6   | eslint, prettier, vitest, tsconfig, cspell configs; `loadConfig` for `.fitnessrc`.                                                                                         | Opinionated defaults; consumer local config wins.                                |
| 7   | Optional `.fitnessrc.ts` / `.fitnessrc.js` — `checks` (names and/or paths), `disabledChecks`.                                                                              | Override order and subset; paths opt in explicitly.                              |
| 8   | Consumer-authored check modules (e.g. `./fitness/checks/*.js`) referenced from `.fitnessrc`.                                                                               | Repo-specific rules without adding a dependency on `@mayjournal/fitness-checks`. |
| 9   | Files checks lint, format, spell-check, or test.                                                                                                                           | Staged paths from git when available.                                            |
| 10  | Dynamic `import()` of check packages from consumer `node_modules`.                                                                                                         | Published checks are normal npm dependencies.                                    |
| 11  | Underlying tools invoked by check implementations.                                                                                                                         | Commodity layer on the Wardley map.                                              |
| 12  | Developer runs CLI from consumer root.                                                                                                                                     | One front door.                                                                  |
| 13  | Agent runs the same CLI via script or hook.                                                                                                                                | Unified enforcement path.                                                        |
| 14  | Runner loads config via `@mayjournal/fitness-shared`.                                                                                                                      | Centralizes config discovery.                                                    |
| 15  | Runner imports the `@mayjournal/fitness-checks/checks/{name}` subpath for name specs.                                                                                      | Runtime loading — not a static registry in the runner.                           |
| 16  | Runner dynamic-imports local modules for path specs in `checks`.                                                                                                           | Same path rules as CLI; runs in-process, not in a worker.                        |
| 17  | When no `checks` list, imports `@mayjournal/fitness-checks/defaultChecks`.                                                                                                 | Bundle defines default run order.                                                |
| 18  | Checks import shared configs and utilities.                                                                                                                                | Avoid duplicating eslint/prettier setup per check.                               |
| 19  | Check `run()` calls tool APIs or subprocesses.                                                                                                                             | Check owns tool-specific behavior.                                               |
| 20  | Checks read consumer tree (staged or full).                                                                                                                                | Validation target is always the app repo.                                        |
| 21  | Local checks live beside app code; default-export a `Check`.                                                                                                               | Same `{ name, run }` contract as npm check packages.                             |
| 22  | Bundle package.json depends on every check package.                                                                                                                        | Install once for full suite.                                                     |
| 23  | Package resolution walks up to nearest install root.                                                                                                                       | Monorepos and nested packages supported.                                         |
| 24  | Shared loader reads `.fitnessrc` from consumer root.                                                                                                                       | Config lives with the app, not the runner source.                                |

## Consumer setup

| Setup                                                | Config                          | What runs                                        |
| ---------------------------------------------------- | ------------------------------- | ------------------------------------------------ |
| `@mayjournal/fitness` + `@mayjournal/fitness-checks` | None                            | Bundle `defaultChecks` in order                  |
| Above + `.fitnessrc` with `checks` (names only)      | Override list                   | Your `checks` order and subset                   |
| Above + `.fitnessrc` with `checks` (names + paths)   | Mixed npm names and local paths | Full configured list in order; paths in-process  |
| Above + `.fitnessrc` with `disabledChecks` only      | Exclude names                   | Bundle list minus disabled                       |
| `@mayjournal/fitness` + `checks` as local paths only | No `@mayjournal/fitness-checks` | Only your local check modules; no bundle install |

All name specs resolve through the installed `@mayjournal/fitness-checks` bundle — there is no separate per-check npm install today. `disabledChecks` applies to npm check names only; local paths are opt-in via explicit `checks` entries and are never removed by it.

`jscpd` is a `defaultChecks` member (bundled dependency, no extra install). `swiftlint` is opt-in only, never in `defaultChecks` — it shells out to a system binary, not an npm package, so a missing install fails clearly rather than crashing.

Install and usage: [README.md](../README.md).
