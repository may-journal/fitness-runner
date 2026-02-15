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
| semantic-commit | Validates HEAD commit message follows Conventional Commits: `type(scope): description` (or `Merge ...`). Allowed types: feat, fix, docs, style, refactor, test, chore. |

## Config

Global config (`.fitnessrc.ts`) and per-file front matter are planned; for now the runner uses safe defaults and the single check (semantic-commit) requires no config.

## Development

```bash
npm install
npm run build
npm run fitness
npm test
```
