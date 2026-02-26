---
relatedConfigurations: ['package.json']
---

# AGENTS.md

## Cursor Cloud specific instructions

This is a Node.js CLI tool (`@fitness/runner`) with no external services, databases, or Docker dependencies. See `README.md` for standard development commands (`npm run build`, `npm run lint`, `npm run format`, `npm test`).

### Non-obvious caveats

- Node.js 24 required. The `.nvmrc` pins Node 24; use `nvm use` to activate it. The update script installs and activates it automatically.
- Build before running the CLI. `npm run build` compiles TypeScript to `dist/`; the CLI entry point is `dist/index.js`. The `npm run fitness` script builds then runs the CLI in one step.
- Commit message format: the `semantic-commit` check enforces `type(scope): description` (scope is required). Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.
- Pre-commit hook runs all fitness checks via `npm run ci`, which builds and executes every check. This takes a few seconds.
- Coverage thresholds are 100%. `npm test` (`vitest run --coverage`) enforces 100% coverage. The entry-point barrel files (`src/index.ts`, `src/runner/index.ts`) show 0% and cause the coverage gate to fail; this is a known pre-existing issue in the repo.
- Lint requires a build step first. `npm run lint` invokes `tsconfig.js --once` before ESLint, which generates `tsconfig.json` from `tsconfig.cjs`. This is handled automatically by the script.
- Markdown files must not use bold (`**`) or italic (`*`) emphasis (enforced by `markdown-no-bold-italic` check).
- All `.md` files need YAML front matter with `fitnessFunctions` or `relatedConfigurations` (enforced by `markdown-front-matter` check).
- When committing, `CHANGELOG.md` must be staged with a new entry whose words overlap the rest of the staged diff (enforced by `changelog-updated` check). New section headings use `### yyyy.mm.dd.HHMM` format with the current UTC time at commit.
