# @fitness/runner

A fitness runner for rules to ensure CI/CD and GenAI are empowered with guardrails of code quality and architectural decision.

## Install

```bash
npm install @fitness/runner
```

## Usage

**CLI** (from repo root):

```bash
npx fitness
```

Run a single check by name:

```bash
npx fitness --check=semantic-commit
```

Run only on staged files (e.g. in a pre-commit hook):

```bash
npx fitness --staged
```

## Checks

| Name  | Description |
|-------|-------------|
| changelog | Validates repo root has `CHANGELOG.md` with at least one dated section (`##` or `###` followed by `yyyy-mm-dd`). |
| node-version | Validates current Node version satisfies `.nvmrc` at repo root (e.g. run `nvm use` if not). |
| semantic-commit | Validates HEAD commit message follows Conventional Commits: `type(scope): description` (or `Merge ...`). Allowed types: feat, fix, docs, style, refactor, test, chore. |

## Config

Optional **`.fitnessrc.ts`** or **`.fitnessrc.js`** at repo root:

```ts
export default {
  checks: ['changelog', 'node-version', 'semantic-commit'], // run these checks, in order
};
```

If present, only listed checks run; if omitted, all checks run. Use `.fitnessrc.js` with `module.exports = { checks: [...] }` for plain Node.

## Checks as sub-projects

Each check lives in **`src/checks/<name>/`** with its implementation, tests, and a README. They are designed to be code-split and a main contribution surface—see [src/checks/README.md](src/checks/README.md) and each check’s folder for behavior and how to add or extend checks.

## Development

```bash
npm install
npm run build
npm run fitness
npm run spell
npm run lint
npm test
```

Spell check uses [cspell](https://cspell.org/) and `cspell.json`; add project words to the `words` array.
