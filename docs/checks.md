---
relatedConfigurations: ['../.fitnessrc.json']
---

# Checks

Every check name has one binary under [go/cmd/](../go/cmd/), with each check's rule documented in its own README (`go/cmd/fitness-check-<name>/README.md`):

- Pure logic: `node-version`, `gitignore-why`, `changelog`, `changelog-updated`, `changelog-bullets`, `semantic-commit`, `commit-attribution`, `plan-trailer`, `read-repo-first`, `markdown-filename-kebab-case`, `markdown-filename-camel-case`, `markdown-front-matter`, `markdown-links`, `markdown-no-bold-italic`, `no-eslint-disable`, `build-output-untracked`, `repeated-string-literals`, `text-readability`, `prose-budget`, `no-plans-dir`, `no-contrastive-reframing`
- Issue and PR body checks: `plan-structure`, `pr-structure`, `pr-closes-issue`, `issue-checklist`; file-level `doc-template` (files match the nearest `*.template.md`) and `requirements` (each Go test owns one)
- `plan-check`, `pr-check`, and `close-check` judge a GitHub blob with body checks, the prose checks included, then an after-effect reacts
- Parsers and network: `issue-link-once` (branch history and the open PR via `gh`), the mermaid diagram/callout checks, `vitest-coverage-exclude`, `dependency-currency` (native npm-registry client)
- Native engines: `cspell` (embedded dictionaries, ~217k words) and `jscpd` (token-based clone detection) — no external tool needed
- `go-complexity`: cyclomatic complexity ceiling for Go, the house eslint rule's counterpart
- Tool wrappers: `prettier`, `eslint`, `vitest-coverage-full`, `swiftlint` — these exec the real tool, resolved from `node_modules/.bin` (walking up) then PATH, never npx
- A check skips when its language or tool is absent; a present language with a missing tool fails with an install hint

`release-changelog` correlates Release Please metadata with the version file and reviewable release notes.

Org mode runs every check in the runner's order ([go/cmd/fitness/main.go](../go/cmd/fitness/main.go)). Each detects whether it applies. External mode also runs all checks by default, with an optional `checks` filter; see [CI setup](ci.md).

## Every tracked file

Checks judge every tracked file; only untracked files stay out, and outside git only `.git`. cspell and jscpd also skip files `.gitattributes` marks `linguist-generated`, such as lock files. A repo setting `ignore`, `skipTheseDirectories`, `proseBudget.exempt`, cspell `ignorePaths`, or a `.prettierignore` fails.

## Shared configs

The opinionated tool configs (eslint flat config, prettier, vitest thresholds, cspell) are embedded inside the check binaries ([go/internal/sharedconf](../go/internal/sharedconf)). They materialize to a cache directory on demand. When a check needs a config, the repo's own config file wins.

Next comes an installed `@mayjournal/fitness-shared` npm package (previously published versions keep working). The embedded copy is the final fallback. The eslint config references plugins that must exist in the consumer repo — exactly the check's peer-tool contract.
