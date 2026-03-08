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

- `@mayjournal/fitness/eslint.config` – ESLint flat config
- `@mayjournal/fitness/vitest.config` – Vitest
- `@mayjournal/fitness/tsconfig` – TypeScript
- `@mayjournal/fitness/cspell` – cspell.json
- `@mayjournal/fitness/prettier.config` – Prettier (semi, singleQuote, tabWidth 2, trailingComma es5, printWidth 100, sort-json for JSON keys); ESLint sort-keys enforces alphabetical object keys in TS/JS/CJS

Example: add `"prettier": "@mayjournal/fitness/prettier.config"` to your package.json to use the shared Prettier config.

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

## How it works

At a high level, the runner decides which checks to run and in what order, builds shared context (e.g. staged files, config), then runs each check. Checks are independent modules that receive `(root, context)` and return pass/fail and optional errors.

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
flowchart LR
  subgraph Runner["Runner"]
    R1["CLI / config"]
    R2["Resolve checks"]
    R3["Build context"]
    R4["Run loop"]
    R1 --> R2 --> R3 --> R4
  end

  subgraph Checks["Checks"]
    C["Registry: each check.run(root, context)"]
  end

  R4 -->|"invoke"| C
  C -->|"pass/fail + errors"| R4
  R4 --> Out["Table + exit code"]

  subgraph Legend["Legend"]
    L_cli["Entry point"]
    L_resolve["Resolve & context"]
    L_execute["Execution"]
  end

  classDef cli fill:#6366f1,stroke:#4f46e5,color:#fff
  classDef resolve fill:#06b6d4,stroke:#0891b2,color:#fff
  classDef execute fill:#10b981,stroke:#059669,color:#fff
  class R1 cli
  class R2,R3 resolve
  class R4,C,Out execute
  class L_cli cli
  class L_resolve resolve
  class L_execute execute
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
    B --> GetChecks["getChecks(argv, root)"]
  end

  GetChecks --> F["resolveCheckSpec(argv)"]
  F --> G{"spec defined?"}
  G -->|no| H["resolveChecks(root)"]
  H --> I{"loadConfig: .fitnessrc.ts / .fitnessrc.js?"}
  I -->|yes, config.checks| J["checksFromConfigList → ordered Check[]"]
  I -->|no or no list| K["registryMinusDisabled(config)"]
  G -->|yes| L["resolveChecksBySpec: registry by name or loadCheckFromPath"]
  J --> M["checks"]
  K --> M
  L --> M
  GetChecks --> E["getStagedContext()<br/><small>git diff --cached → stagedFiles</small>"]
  M --> PassthroughArgs["Single check: getInlineContextFragment + getPassthroughArgs"]
  PassthroughArgs --> N["buildContext(staged, inlineFragment, checks, passthroughArgs)"]
  E --> N
  M --> N
  N --> O["runChecks(checks, root, context)"]
  M --> O

  subgraph Execute["Execute checks"]
    O --> P["collectResults: for each check in order"]
    P --> Q["runOneCheck (worker or in-process)<br/>check.run(root, context)"]
    Q --> R{"result.ok?"}
    R -->|yes| S["Push result, continue"]
    R -->|no| T["Push result, count failure"]
    S --> P
    T --> P
  end

  Execute --> V["buildTable(results), buildTotalLine(counts)"]
  V --> U["process.exit(failed ? 1 : 0)"]

  subgraph Legend["Legend"]
    L_cli["Entry point"]
    L_resolve["Resolve & context"]
    L_execute["Execution"]
    L_decision{"Branch?"}
  end

  linkStyle 3,5,6,7,21,22 stroke:#64748b,color:#1e293b
  classDef cli fill:#6366f1,stroke:#4f46e5,color:#fff
  classDef resolve fill:#06b6d4,stroke:#0891b2,color:#fff
  classDef execute fill:#10b981,stroke:#059669,color:#fff
  classDef decision fill:#f1f5f9,stroke:#64748b,color:#334155
  class A,B cli
  class GetChecks,PassthroughArgs,E,F,G,H,I,J,K,L,M,N resolve
  class O,P,Q,R,S,T,V,U execute
  class G,I decision
  class L_cli cli
  class L_resolve resolve
  class L_execute execute
  class L_decision decision
```
