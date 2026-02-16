---
relatedConfigurations: ['../../package.json']
---

# Checks

Each check is a self-contained sub-project in its own folder:

- [read-repo-first](./read-repo-first/) – CLI reminder to familiarize with repo decisions and enabled checks
- [changelog](./changelog/) – Root `CHANGELOG.md` with dated sections
- [changelog-updated](./changelog-updated/) – When staged context is present, changelog must mention words from staged diff
- [cspell](./cspell/) – Runs cspell for spell-checking (skips if no `cspell.json`)
- [eslint](./eslint/) – Runs ESLint; staged files or full repo
- [markdown-no-bold-italic](./markdown-no-bold-italic/) – No bold/italic emphasis in `.md` files unless required
- [node-version](./node-version/) – Node version satisfies `.nvmrc`
- [rules-front-matter](./rules-front-matter/) – Markdown front matter `fitnessFunctions`/`relatedConfigurations` paths must exist
- [semantic-commit](./semantic-commit/) – HEAD commit follows Conventional Commits
- [vitest-coverage-exclude](./vitest-coverage-exclude/) – Vitest coverage `exclude` must not list any `.ts` file or pattern

Each folder contains the check implementation (`index.ts`), tests (`*.test.ts`), and a README describing behavior and how to contribute. The runner registry in `index.ts` imports these and runs them in order (or per `.fitnessrc`).

## Adding a check

1. Create a new folder under `src/checks/<name>/`.
2. Export a `Check` from `index.ts` (see existing checks for the shape).
3. Add a README and tests next to the implementation.
4. Register the check in `src/checks/index.ts` and in `.fitnessrc.ts` if you use config.

Checks are a major contribution point: they are code-split by folder and easy to add or maintain independently.
