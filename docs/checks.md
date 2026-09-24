---
relatedConfigurations: ['../.fitnessrc.json']
---

# Checks

Every check name has one binary under [go/cmd/](../go/cmd/), with each check's rule documented in its own README (`go/cmd/fitness-check-<name>/README.md`):

- Pure logic: `node-version`, `gitignore-why`, `changelog`, `changelog-updated`, `changelog-bullets`, `semantic-commit`, `commit-attribution`, `read-repo-first`, `markdown-filename-kebab-case`, `markdown-filename-camel-case`, `markdown-front-matter`, `markdown-links`, `markdown-no-bold-italic`, `no-eslint-disable`, `build-output-untracked`, `repeated-string-literals`, `text-readability`, `prose-budget`
- Body checks (run as GitHub Actions on Issue and PR bodies): `plan-structure`, `pr-structure`
- Parsers and network: the five mermaid diagram/callout checks, `vitest-coverage-exclude`, `dependency-currency` (native npm-registry client)
- Native engines: `cspell` (embedded dictionaries, ~217k words) and `jscpd` (token-based clone detection) — no external tool needed
- `go-complexity`: cyclomatic complexity ceiling for Go, the house eslint rule's counterpart
- Tool wrappers: `prettier`, `eslint`, `vitest-coverage-full`, `swiftlint` — these exec the real tool, resolved from `node_modules/.bin` (walking up) then PATH, never npx
- A missing tool binary fails with a one-line install hint

Default run order lives in the runner ([go/cmd/fitness/main.go](../go/cmd/fitness/main.go)). Opt-in checks (`swiftlint`, `commit-attribution`, the mermaid family, and others) are enabled per repo via `.fitnessrc.json`.

## Shared configs

The opinionated tool configs (eslint flat config, prettier, vitest thresholds, cspell) are embedded inside the check binaries ([go/internal/sharedconf](../go/internal/sharedconf)). They materialize to a cache directory on demand. When a check needs a config, the repo's own config file wins.

Next comes an installed `@mayjournal/fitness-shared` npm package (previously published versions keep working). The embedded copy is the final fallback. The eslint config references plugins that must exist in the consumer repo — exactly the check's peer-tool contract.
