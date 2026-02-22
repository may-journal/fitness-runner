---
# Top-level project config
relatedConfigurations: ['package.json']
---

# @fitness/runner

Node fitness runner that runs checks for local dev, CI/CD, and GenAI workflows to stay aligned with your intended rules and quality bar.

## Install

```bash
npm install @fitness/runner
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

## Checks

See [src/checks/README.md](src/checks/README.md) for the list and how checks work. Each check has its own README:

- [read-repo-first](src/checks/read-repo-first/README.md)
- [changelog](src/checks/changelog/README.md)
- [changelog-updated](src/checks/changelog-updated/README.md)
- [cspell](src/checks/cspell/README.md)
- [eslint](src/checks/eslint/README.md)
- [markdown-no-bold-italic](src/checks/markdown-no-bold-italic/README.md)
- [prettier](src/checks/prettier/README.md)
- [node-version](src/checks/node-version/README.md)
- [rules-front-matter](src/checks/rules-front-matter/README.md)
- [semantic-commit](src/checks/semantic-commit/README.md)
- [vitest-coverage-exclude](src/checks/vitest-coverage-exclude/README.md)

## Exported configs

Consumers can extend these configs via package exports:

- `@fitness/runner/eslint.config` – ESLint flat config
- `@fitness/runner/vitest.config` – Vitest
- `@fitness/runner/tsconfig` – TypeScript
- `@fitness/runner/cspell` – cspell.json
- `@fitness/runner/prettier.config` – Prettier (semi, singleQuote, tabWidth 2, trailingComma es5, printWidth 100, sort-json for JSON keys); ESLint sort-keys enforces alphabetical object keys in TS/JS/CJS

Example: add `"prettier": "@fitness/runner/prettier.config"` to your package.json to use the shared Prettier config.

## Config

Optional `.fitnessrc.ts` or `.fitnessrc.js` at repo root:

```ts
export default {
  checks: ['changelog', 'node-version', 'semantic-commit'], // run these checks, in order
};
```

If present, only listed checks run; if omitted, all checks run. Use `.fitnessrc.js` with `module.exports = { checks: [...] }` for plain Node.

## Checks as sub-projects

Each check lives in `src/checks/<name>/` with its implementation, tests, and a README. They are designed to be code-split and a main contribution surface—see [src/checks/README.md](src/checks/README.md) and each check’s folder for behavior and how to add or extend checks.

## Development

```bash
npm install
npm run build
npm run fitness
npm run format
npm run lint
npm test
```

## Code flow and check process

```mermaid
%%{init: {'theme':'base', 'themeVariables': {
  'primaryColor':'#6366f1',
  'primaryTextColor':'#fff',
  'primaryBorderColor':'#4f46e5',
  'secondaryColor':'#06b6d4',
  'secondaryTextColor':'#fff',
  'secondaryBorderColor':'#0891b2',
  'tertiaryColor':'#10b981',
  'tertiaryTextColor':'#fff',
  'tertiaryBorderColor':'#059669',
  'lineColor':'#64748b',
  'background':'#f8fafc',
  'mainBkg':'#e0e7ff',
  'clusterBkg':'#f1f5f9',
  'clusterBorder':'#94a3b8',
  'titleColor':'#334155',
  'textColor':'#1e293b',
  'labelColor':'#1e293b',
  'edgeLabelBackground':'#f8fafc',
  'nodeTextColor':'#1e293b',
  'fontFamily':'system-ui, sans-serif'
}}}%%
flowchart TD
  subgraph CLI["CLI entry"]
    A["npx fitness (argv)"]
    A --> B["run(argv)"]
  end

  B --> F["resolveCheckSpec(argv)"]
  F --> G{"spec defined?"}
  G -->|no| H["resolveChecks(root)"]
  H --> I{".fitnessrc.ts / .fitnessrc.js exists?"}
  I -->|yes| J["config.checks → ordered Check[]"]
  I -->|no| K["Full registry (all checks)"]
  G -->|yes| L["resolveChecksBySpec(spec, root) → one check by name or path"]
  J --> M["checks"]
  K --> M
  L --> M
  B --> E["getStagedContext()<br/><small>git diff --cached → stagedFiles</small>"]
  E --> N["buildContext(staged, inlineFragment, checks, passthrough)"]
  M --> Dctx["When single check: inlineFragment + passthrough from check.contextInline"]
  Dctx --> N
  M --> N
  N --> O["runChecks(checks, root, context)"]
  M --> O

  subgraph Execute["Execute checks"]
    O --> P["For each check in order"]
    P --> Q["runOneCheck: check.run(root, context)"]
    Q --> R{"result.ok?"}
    R -->|yes| S["Log meta, continue"]
    R -->|no| T["Set failed, collect errors for table"]
    S --> P
    T --> P
  end

  Execute --> U["process.exit(failed ? 1 : 0)"]

  linkStyle 3,5,6,7,21,22 stroke:#64748b,color:#1e293b
  classDef cli fill:#6366f1,stroke:#4f46e5,color:#fff
  classDef resolve fill:#06b6d4,stroke:#0891b2,color:#fff
  classDef execute fill:#10b981,stroke:#059669,color:#fff
  classDef decision fill:#f1f5f9,stroke:#64748b,color:#334155
  class A,B cli
  class Dctx,E,F,G,H,I,J,K,L,M,N resolve
  class O,P,Q,R,S,T,U execute
  class G,I decision
```
