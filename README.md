---
# Top-level project config
relatedConfigurations: ["package.json"]
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
npx fitness semantic-commit
# or
npx fitness --check=semantic-commit
```

Commit-msg hook: pass the message file as a positional so semantic-commit validates the proposed message: `fitness --check=semantic-commit "$1"`.

## Checks

See [src/checks/README.md](src/checks/README.md) for the list and how checks work. Each check has its own README:

- [changelog](src/checks/changelog/README.md)
- [changelog-updated](src/checks/changelog-updated/README.md)
- [cspell](src/checks/cspell/README.md)
- [node-version](src/checks/node-version/README.md)
- [rules-front-matter](src/checks/rules-front-matter/)
- [semantic-commit](src/checks/semantic-commit/README.md)

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
  'edgeLabelBackground':'#f8fafc',
  'nodeTextColor':'#1e293b',
  'fontFamily':'system-ui, sans-serif'
}}}%%
flowchart TD
  subgraph CLI["CLI entry"]
    A["npx fitness (argv)"]
    A --> B["run(argv)"]
  end

  B --> F["resolveCheckName(argv)"]
  F --> G{".fitnessrc.ts / .fitnessrc.js exists?"}
  G -->|yes| H["config.checks → ordered Check[]"]
  G -->|no| I["Full registry (all checks)"]
  H --> J["--check=name or positional → one or all"]
  I --> J
  J --> K["runChecks(checks, root, context)"]
  B --> E["getStagedContext()"]
  B --> Dctx["getCommitMsgContext if --check=semantic-commit + positional path"]
  E --> K
  Dctx --> K

  subgraph Execute["Execute checks"]
    K --> L["For each check in order"]
    L --> M["runOneCheck: check.run(root, context)"]
    M --> N{"result.ok?"}
    N -->|yes| O["Log meta, continue"]
    N -->|no| P["Set failed, collect errors for table"]
    O --> L
    P --> L
  end

  Execute --> Q["process.exit(failed ? 1 : 0)"]

  classDef cli fill:#6366f1,stroke:#4f46e5,color:#fff
  classDef resolve fill:#06b6d4,stroke:#0891b2,color:#fff
  classDef execute fill:#10b981,stroke:#059669,color:#fff
  classDef decision fill:#f1f5f9,stroke:#64748b,color:#334155
  class A,B cli
  class Dctx,E,F,G,H,I,J resolve
  class K,L,M,N,O,P,Q execute
  class G decision
```
