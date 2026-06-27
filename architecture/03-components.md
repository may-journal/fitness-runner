---
relatedConfigurations: ['../package.json']
---

# Components

```mermaid
C4Component
    title Runner CLI

    Person(developer, "1 Developer")

    System_Boundary(cli, "Runner CLI") {
        Container_Boundary(entry, "Entry") {
            Component(main, "2 CLI entry", "Node")
        }
        Container_Boundary(runner, "Run loop") {
            Component(resolve, "3 Check resolver", "run-resolve")
            Component(loader, "4 Check loader", "load-check")
            Component(execute, "5 Check executor", "run-execute")
            Component(output, "6 Results output", "run-output")
        }
    }

    System_Boundary(checks, "Check package") {
        Component(check, "7 Check module", "default export")
    }

    System_Boundary(shared, "Shared") {
        Component(config, "8 Config loader", "loadConfig")
    }

    System_Ext(localCheck, "21 Local check module")
    System_Ext(bundle, "9 defaultChecks")
    System_Ext(git, "10 git")

    Rel(developer, main, "11")
    Rel(main, resolve, "12")
    Rel(resolve, loader, "13")
    Rel(resolve, config, "14")
    Rel(loader, bundle, "15")
    Rel(loader, check, "16")
    Rel(loader, localCheck, "22")
    Rel(resolve, git, "17")
    Rel(main, execute, "18")
    Rel(execute, check, "19")
    Rel(execute, localCheck, "23")
    Rel(main, output, "20")
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

| #   | Description                                                                 | Why                                                       |
| --- | --------------------------------------------------------------------------- | --------------------------------------------------------- |
| 1   | Developer or agent invokes `fitness`.                                       | Same entry as system context.                             |
| 2   | `packages/runner/src/index.ts` — calls `run()` when main module.            | Thin entry; logic lives in runner modules.                |
| 3   | `getChecks` — argv parsing, single-check mode, context assembly.            | One place for spec resolution (name, path, full list).    |
| 4   | `resolveCheckNames`, `resolveCheckSpecs`, `loadCheck`, `loadCheckFromPath`. | Names → npm packages; paths → consumer modules.           |
| 5   | `runOneCheck` — worker thread or in-process with timeout.                   | Isolates slow checks; path-loaded checks stay in-process. |
| 6   | Results table, totals, chalk formatting, exit code.                         | User-visible pass/fail summary.                           |
| 7   | Each check's `Check` default export — `name`, `run()`.                      | Contract every package must satisfy.                      |
| 8   | Reads `.fitnessrc`; `checks` accepts names and/or relative paths.           | Shared between resolver and loader.                       |
| 9   | `@mayjournal/fitness-checks/defaultChecks` when config has no `checks`.     | Ordered default list.                                     |
| 10  | `git diff --cached --name-only` for staged file context.                    | Pre-commit and partial runs.                              |
| 11  | User-facing invocation.                                                     |                                                           |
| 12  | `run()` delegates to resolver first.                                        | Resolve before execute.                                   |
| 13  | Resolver asks loader for check modules by spec (name or path).              | Separation of argv/config from import mechanics.          |
| 14  | Resolver loads fitness config for the check list.                           | `.fitnessrc` drives order and subset.                     |
| 15  | Loader imports bundle subpath when needed.                                  | Default name list without hardcoding in runner.           |
| 16  | Loader dynamic-imports `@mayjournal/fitness-check-{name}`.                  | Throws if missing or `default.name` mismatch.             |
| 17  | Resolver builds `stagedFiles` in `RunContext`.                              | Checks receive consistent context.                        |
| 18  | Run loop calls executor per check.                                          | Sequential run with aggregated results.                   |
| 19  | Executor calls `check.run(root, context)` for npm checks.                   | Check owns tool invocation.                               |
| 20  | Run loop prints table after all checks finish.                              | Single summary per invocation.                            |
| 21  | Consumer module default-exporting `{ name, run }` — same as npm checks.     | Repo-specific checks without a published package.         |
| 22  | Loader resolves path specs relative to project root; marks path-loaded.     | Config and CLI share one code path.                       |
| 23  | Executor runs path-loaded checks in-process, not via worker.                | Local modules may not be worker-safe.                     |

## Resolve check specs (priority)

`resolveCheckNames` in `packages/runner/src/checks/load-check.ts` produces an ordered spec list; `resolveCheckSpecs` loads each entry:

1. `.fitnessrc` with `checks` — use that list (order preserved). Each entry is either:
   - Name spec — `@mayjournal/fitness-check-{name}` via `loadCheck`.
   - Path spec — relative module path via `loadCheckFromPath` (contains `/` or `\`, or ends in `.js` / `.mjs` / `.cjs` / `.ts`).
2. No `checks` list — import `defaultChecks` from `@mayjournal/fitness-checks/defaultChecks` (names only).
3. `disabledChecks` — remove matching name specs from the list from step 1 or 2. Path specs are unchanged (opt-in only).
4. Neither step 1 nor 2 available — throw: install `@mayjournal/fitness-checks` or set `checks` in `.fitnessrc`.

Dedupe (first occurrence wins): name specs by check name; path specs by resolved absolute path; if a loaded path check's `name` matches an earlier entry, skip the duplicate.

Explicit `checks` with missing entries: skip uninstalled name specs (`allowMissing`); fail on missing or invalid path specs (path was explicitly configured).

Single-check mode bypasses the list: `npx fitness prettier`, `npx fitness --check=eslint`, or `npx fitness --check=./my-check.mjs`.

## Run context

| Field                  | Source                                      |
| ---------------------- | ------------------------------------------- |
| `enabledCheckNames`    | names in this run                           |
| `registeredCheckNames` | same as enabled                             |
| `checkFolderByName`    | each loaded `Check.folder` when set         |
| `stagedFiles`          | git staged paths (node_modules stripped)    |
| `passthroughArgs`      | args after check spec for single-check runs |

Built from loaded checks and the runner — not a separate registry.
