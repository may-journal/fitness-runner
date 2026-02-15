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
| semantic-commit | Validates HEAD commit message follows Conventional Commits: `type(scope): description` (or `Merge ...`). Allowed types: feat, fix, docs, style, refactor, test, chore. |

## Config

Optional **`.fitnessrc.ts`** or **`.fitnessrc.js`** at repo root:

```ts
export default {
  checks: ['changelog', 'semantic-commit'], // run these checks, in order
};
```

If present, only listed checks run; if omitted, all checks run. Use `.fitnessrc.js` with `module.exports = { checks: [...] }` for plain Node.

## Development

```bash
npm install
npm run build
npm run fitness
npm test
```
