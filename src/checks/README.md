---
relatedConfigurations: ['../../package.json']
---

# Checks

Each check is a self-contained sub-project in its own folder:

- [read-repo-first](./read-repo-first/) – CLI reminder to familiarize with repo decisions and enabled checks
- [changelog](./changelog/) – Root `CHANGELOG.md` with dated sections
- [changelog-updated](./changelog-updated/) – When staged context is present, changelog must mention words from staged diff
- [cspell](./cspell/) – Spell-checking via cspell (uses package `cspell.json` when the project has none)
- [eslint](./eslint/) – ESLint via this package’s config; no local ESLint or tsconfig required
- [markdown-no-bold-italic](./markdown-no-bold-italic/) – No bold/italic emphasis in `.md` files unless required
- [prettier](./prettier/) – Prettier `--check` (uses package config when the project has none); staged files or full repo
- [node-version](./node-version/) – Node version satisfies `.nvmrc`
- [rules-front-matter](./rules-front-matter/) – Markdown front matter `fitnessFunctions`/`relatedConfigurations` paths must exist
- [semantic-commit](./semantic-commit/) – HEAD commit follows Conventional Commits
- [vitest-coverage-exclude](./vitest-coverage-exclude/) – Vitest coverage `exclude` must not list disallowed `.ts` patterns (uses package vitest config when the project has none)
- [vitest-coverage-full](./vitest-coverage-full/) – Requires 100% coverage thresholds and `vitest run --coverage` pass (uses package vitest config when the project has none)

Each folder contains the check implementation (`index.ts`), tests (`*.test.ts`), and a README describing behavior and how to contribute. The runner registry in `index.ts` imports these and runs them in order (or per `.fitnessrc`).

## Adding a check

1. Create a new folder under `src/checks/<folder>/`. Convention: folder name equals the check name (e.g. `src/checks/my-check/` for name `my-check`).
2. Export a `Check` from `index.ts` with `name`, `run`, and optional `folder` (set `folder` only when the folder name differs from the check name, e.g. `rules-front-matter` for `markdown-front-matter`).
3. Add a README and tests next to the implementation.
4. Add the check name to the `CheckName` enum in `src/types/check-name.ts`.
5. Import the check and add it to the `registry` array in `src/checks/index.ts`.

The test in `src/checks/registry.test.ts` ensures the enum and registry stay in sync (every registry entry has an enum value and vice versa).

Checks are a major contribution point: they are code-split by folder and easy to add or maintain independently.
