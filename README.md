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

See [Architecture.md](./Architecture.md) for the C4 model ([architecture/](architecture/)).

## Checks

Each check is an npm package under `packages/checks/<name>/` (`@mayjournal/fitness-check-<name>`) with its own README. Default run order when `.fitnessrc` omits `checks` is `defaultChecks` from `@mayjournal/fitness-checks` (see [packages/checks-bundle/src/index.ts](./packages/checks-bundle/src/index.ts) and [Architecture.md](./Architecture.md)).

## Consumer projects

Install the runner plus either the checks bundle or individual check packages.

Bundle (recommended): install `@mayjournal/fitness` and `@mayjournal/fitness-checks`. With no `.fitnessrc`, the runner uses bundle `defaultChecks`. Optional `.fitnessrc` can set `checks` to override the list or `disabledChecks` to exclude names from the bundle default.

À la carte: install `@mayjournal/fitness` and only the `@mayjournal/fitness-check-*` packages you need; set `checks` in `.fitnessrc` to that subset (each listed name must be installed).

Add a script and run from your repo root. Checks use shared configs automatically—you do not need local copies of `eslint.config`, `prettier.config`, `vitest.config`, `tsconfig`, or `cspell.json`. Setup matrix and examples: [Architecture.md](./Architecture.md#consumer-setup).

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

### Git hooks (optional)

`@mayjournal/fitness` ships starter hooks under `githooks/`. Copy them into your repo, then point Git at that folder once:

```bash
cp -a node_modules/@mayjournal/fitness/githooks githooks
git config core.hooksPath githooks
```

Hooks run `npm run fitness` (pre-commit) and semantic-commit validation (commit-msg). Edit under `githooks/` after copy.

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
