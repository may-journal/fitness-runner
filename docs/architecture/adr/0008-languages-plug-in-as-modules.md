---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0008 — Languages plug in as modules through one registry

Check: [fitness/adr-0008](../../../fitness/adr-0008/README.md)

## Context

Language knowledge was copied through the runner: hand lists of checks, lockfiles and dictionaries, and readers inside shared packages. Each new language added another copy.

## Decision

1. Each language is one `lang.Module` value in its own file under `go/internal/lang/langs`, and the registry orders them as check rows print.
2. A module brings data and hooks: endings, manifests, lockfiles, build output, tools, checks, dictionaries, and runtime, view and test readers.
3. A language's own code and prose live in `go/internal/lang/<language>`. Shared code and shared docs name no language.
4. Lists are discovered, never kept by hand: installed check binaries, embedded dictionaries, and module claims. What no module claims is repo-wide.
5. An exclusion, or a language with no module, is one commented file in a data folder: `ignores`, `sources` or `lockfiles`.
6. The views read code through one `codemodel`. A module's `Views` hook fills it, and token readers share one walk that relates names.
7. When a language needs something shared code lacks, the shared code changes for every language.
8. Rejected: Go `plugin` loading, which needs cgo and one toolchain for runner and plugins.

## Consequences

1. A new language is one module file and its own folder; repo-wide checks pick it up with no edits.
2. Check names, `--checks` and `.fitnessrc.json` keys stay as they were.
3. A language that needs a hook the module type lacks widens the type for every module.
4. The registry is compiled in, so adding a language still needs a release.
