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

## Config

Optional `.fitnessrc.ts` or `.fitnessrc.js` at repo root:

```ts
export default {
  checks: ['changelog', 'node-version', 'semantic-commit'], // run these checks, in order
  disabledChecks: ['cspell'], // optional: exclude from bundle defaultChecks
};
```

If `checks` is set, only those checks run (in order). If omitted, the runner uses `defaultChecks` from `@mayjournal/fitness-checks` (install the bundle package). `disabledChecks` removes names from either list. Use `.fitnessrc.js` with `module.exports = { checks: [...] }` for plain Node.

`checks` entries can also be local paths, mixed in with npm check names, to run a repo-specific check without publishing a package:

```ts
// ./fitness/checks/no-console.js
export default {
  name: 'no-console',
  run: async () => ({ ok: true, errors: [] }),
};
```

```ts
// .fitnessrc.ts
export default {
  checks: ['cspell', './fitness/checks/no-console.js'], // npm name + local path, in order
};
```

A local check module default-exports (or named-exports) an object with `name` and `run` — same shape as a published check. A missing or invalid path fails the run with an error, since it was explicitly configured; `disabledChecks` cannot remove path entries.

### Duplicate-code detection

`jscpd` is part of `defaultChecks` — it's bundled as a dependency, no extra install. It fails when duplicated lines exceed 1% of the codebase; see [packages/checks/jscpd](packages/checks/jscpd) for flags and the `jscpd:ignore-start`/`jscpd:ignore-end` escape hatch for justified duplication.

### Opt-in: SwiftLint

`swiftlint` is not part of `defaultChecks` — most repos have no Swift code. Add it explicitly:

```ts
export default {
  checks: ['cspell', 'swiftlint'],
};
```

It shells out to a system binary (`brew install swiftlint`) — it isn't an npm package, so unlike every other check it isn't bundled; a missing binary fails clearly instead of crashing, and a repo with no Swift files passes clean. See [packages/checks/swiftlint](packages/checks/swiftlint) for behavior.

## Git hooks

This is the point of the runner: checks enforced automatically on `git commit`, not something a developer has to remember to run. `@mayjournal/fitness` ships starter hooks under `githooks/`; copy them into your repo and point Git at that folder once:

```bash
cp -a node_modules/@mayjournal/fitness/githooks githooks
git config core.hooksPath githooks
```

Hooks run `npm run fitness` (pre-commit) and semantic-commit validation (commit-msg). Edit under `githooks/` after copy.

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

See [architecture-index.md](./architecture-index.md) for the C4 model ([architecture/](architecture/)).

## Checks

Each check's source lives under `packages/checks/<name>/` with its own README. Consumers load a check by name via the `@mayjournal/fitness-checks/checks/<name>` subpath — the individual `@mayjournal/fitness-check-<name>` workspaces are private and never published on their own. Default run order when `.fitnessrc` omits `checks` is `defaultChecks` from `@mayjournal/fitness-checks` (see [packages/checks-bundle/src/index.ts](./packages/checks-bundle/src/index.ts) and [architecture-index.md](./architecture-index.md)).

## Consumer projects

Install `@mayjournal/fitness` and `@mayjournal/fitness-checks`. With no `.fitnessrc`, the runner uses bundle `defaultChecks`. Optional `.fitnessrc` can set `checks` to run a subset (still resolved from the installed bundle, in your order) or `disabledChecks` to exclude names from the bundle default.

To add a repo-specific rule without a new dependency, skip the name and point `checks` at a local module path instead — see [Config](#config) above.

Add a script and run from your repo root. Checks use shared configs automatically—you do not need local copies of `eslint.config`, `prettier.config`, `vitest.config`, `tsconfig`, or `cspell.json`. Setup matrix and examples: [architecture-index.md](./architecture-index.md#consumer-setup).

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

## Go runner

The suite has been rebuilt in Go per [plans/archive/01-go-rewrite.md](./plans/archive/01-go-rewrite.md): one static zero-dependency binary per check plus a `fitness` runner binary, all under [go/](go/). All 27 check names are ported with side-by-side parity against the TypeScript checks (`npm run parity:go` diffs every check's ok/errors/filesChecked on this repo).

This repo now gates its own commits on the Go suite: `npm run fitness` builds the binaries (`npm run build:go`, ~300ms warm) and runs `go/bin/fitness`, driven by `.fitnessrc.json`. The full 23-check dogfood suite runs in under 2 seconds — the TypeScript suite (`npm run fitness:ts`, still fully supported and CI-gated during the transition) takes ~6.5s plus a build.

Tool-wrapper checks (eslint, prettier, cspell natively reimplemented; vitest and swiftlint executed as peer tools) resolve peer binaries from `node_modules/.bin` walking up, then PATH — never npx — and fail with one-line install hints when missing.

Distribution (decided): GitHub Releases with prebuilt static binaries plus `go install`, once the TypeScript retirement question is settled; an npm shim that fetches the platform binary can follow if consumers want `npx fitness` continuity. The npm TypeScript packages remain published and unchanged until then.

## Development

```bash
npm install
npm run build
npm run fitness
npm run format
npm run lint
npm test
```

`npm install` runs `prepare`, which points Git at this repo's `githooks/` (pre-commit runs `npm run fitness` + lint; commit-msg validates semantic-commit). If hooks aren't firing — e.g. `core.hooksPath` got reset — re-run:

```bash
git config core.hooksPath githooks
```
