---
# Top-level project config
relatedConfigurations: ['package.json']
---

# @mayjournal/fitness

Node fitness runner that runs checks for local dev, CI/CD, and GenAI workflows to stay aligned with your intended rules and quality bar.

## Install

```bash
npm install @mayjournal/fitness
```

## Usage

CLI (from repo root):

```bash
npx fitness
```

Run a single check by name (positional or flag; use `--` before flags if npx swallows them):

```bash
npx fitness prettier
npx fitness prettier --write .
# or
npx fitness semantic-commit
# or
npx fitness --check=semantic-commit
```

Checks can register `contextInline` so the runner injects a named arg value into context and strips it from passthrough. For the commit-msg hook with semantic-commit, pass the message string: `fitness --check=semantic-commit --message="$(cat "$1")"`. See each check’s README for its arg name.

See [Architecture.md](./Architecture.md) for monorepo layout and runtime flow.

## Checks

Each check is an npm package under `packages/checks/<name>/` with its own README:

- [read-repo-first](packages/checks/read-repo-first/README.md)
- [changelog](packages/checks/changelog/README.md)
- [changelog-updated](packages/checks/changelog-updated/README.md)
- [cspell](packages/checks/cspell/README.md)
- [eslint](packages/checks/eslint/README.md)
- [markdown-no-bold-italic](packages/checks/markdown-no-bold-italic/README.md)
- [prettier](packages/checks/prettier/README.md)
- [node-version](packages/checks/node-version/README.md)
- [markdown-front-matter](packages/checks/markdown-front-matter/README.md)
- [semantic-commit](packages/checks/semantic-commit/README.md)
- [vitest-coverage-exclude](packages/checks/vitest-coverage-exclude/README.md)
- [vitest-coverage-full](packages/checks/vitest-coverage-full/README.md)

## Consumer projects

Install `@mayjournal/fitness` and `@mayjournal/fitness-checks` (bundle of all checks), add a script, and run from your repo root. Checks use shared configs automatically—you do not need local copies of `eslint.config`, `prettier.config`, `vitest.config`, `tsconfig`, or `cspell.json`. See [Architecture.md](./Architecture.md) for à la carte setup.

```json
{
  "dependencies": {
    "@mayjournal/fitness": "...",
    "@mayjournal/fitness-checks": "..."
  },
  "scripts": {
    "fitness": "fitness"
  }
}
```

```bash
npm run fitness
```

If your project already has its own config for a tool, that local file wins; otherwise fitness falls back to bundled configs from `@mayjournal/fitness-shared` (installed as a dependency of `@mayjournal/fitness`).

## Exported configs (optional)

Install `@mayjournal/fitness-shared` if you want to wire these tools directly (outside `npm run fitness`). Fitness checks use the same exports internally:

- `@mayjournal/fitness-shared/eslint.config` – ESLint flat config
- `@mayjournal/fitness-shared/vitest.config` – Vitest
- `@mayjournal/fitness-shared/tsconfig` – TypeScript (this repo’s build config)
- `@mayjournal/fitness-shared/cspell` – cspell.json
- `@mayjournal/fitness-shared/prettier.config` – Prettier (semi, singleQuote, tabWidth 2, trailingComma es5, printWidth 100, sort-json for JSON keys); ESLint sort-keys enforces alphabetical object keys in TS/JS/CJS

## Config

Optional `.fitnessrc.ts` or `.fitnessrc.js` at repo root:

```ts
export default {
  checks: ['changelog', 'node-version', 'semantic-commit'], // run these checks, in order
  disabledChecks: ['cspell'], // optional: exclude from bundle defaultChecks
};
```

If `checks` is set, only those checks run (in order). If omitted, the runner uses `defaultChecks` from `@mayjournal/fitness-checks` (install the bundle package). `disabledChecks` removes names from either list. Use `.fitnessrc.js` with `module.exports = { checks: [...] }` for plain Node.

## Check packages

Each check lives in `packages/checks/<name>/` as `@mayjournal/fitness-check-<name>`. See [Architecture.md](./Architecture.md) for how the runner loads them and each check’s README for behavior.

## Development

```bash
npm install
npm run build
npm run fitness
npm run format
npm run lint
npm test
```
