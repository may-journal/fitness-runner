---
relatedConfigurations: ['../.fitnessrc.json']
---

# Checks

Every check name has one binary under [go/cmd/](../go/cmd/), with each check's rule documented in its own README (`go/cmd/fitness-check-<name>/README.md`). The [fitness skill](../skills/fitness/SKILL.md) shows how to read them:

- Pure logic: rules over file names, text, commits, and plans, such as `gitignore-why`, `semantic-commit`, `markdown-links`, and `prose-budget`
- Issue and PR body checks: `plan-structure`, `epic-structure`, `pr-structure`, `pr-closes-issue`, `issue-checklist`; file-level `doc-template` (files match their nearest template) and `requirements` (tests change with docs)
- `issue-check`, `pr-check`, `close-check`, and `comment-check` judge a GitHub blob with the body and prose checks, then an after-effect reacts; [`fitness --blob`](check-protocol.md#blob-input) judges any text
    - `issue-check` picks each issue's checks by its kind label and `.github/ISSUE_TEMPLATE`; `plan-check` is its Plan-only alias
    - `dupe-check` runs no check; it notes when a new issue's words repeat an older issue's
- Parsers and network: the `architecture-*` checks, `issue-link-once`, `workflow-ref-exists`, and the mermaid diagram/callout checks
- Native engines: `cspell` (embedded dictionaries, ~217k words) and `jscpd` (token-based clone detection) — no external tool needed
- Language checks: each [language module](../go/internal/lang/langs/langs.go) claims its own checks, and its tool wrappers run the real tool the way the module finds it
- A check skips when its language or tool is absent; a present language with a missing tool fails with an install hint

`release-changelog` correlates Release Please metadata with the version file. `pr-check` holds a PR to one [entry file](../go/cmd/fitness-check-changelog/README.md#entry-files) named for its heading's date, naming extras and a bad name. The description must link the entry the PR adds and never copy a bullet it adds. No check reads `CHANGELOG.md`; each release's notes come from its entry files.

Org mode runs every check in the runner's order ([go/cmd/fitness/main.go](../go/cmd/fitness/main.go)). Each detects whether it applies. External mode also runs all checks by default, with an optional `checks` filter; see [CI setup](ci.md). A repo can add its own [local checks](check-protocol.md#local-checks), and an external repo can [disable checks](check-protocol.md#disabled-checks).

## List the checks

`fitness list` names every check this repo can run, built-in or local, in run order. Each row gives the check's summary, its scope, its time budget, and whether the repo's config enables it. `fitness list --json` gives the same fields for an agent; `fitness check --list` still prints the bare built-in names.

A scope, declared in the check's `checkkit.Describe`, says what it reads: `files`, one given `blob` of text, `git` history, or live `github` metadata. The summary is the first sentence of the check's Go package doc comment, which `go generate` writes into the runner. The local [generated check](../checks/README.md) fails when that file is stale.

`fitness explain <check>` prints the check's whole package doc comment, rendered as Markdown, and the `.fitnessrc.json` keys it reads. A config struct field tagged `check:"<name>"` names such a key, and its doc comment explains it. `--json` gives the name, summary, scope, budget, and body.

## Every tracked file

Checks judge every tracked file; only untracked files stay out, and outside git only `.git`. cspell and jscpd also skip files `.gitattributes` marks `linguist-generated`, such as lock files. A repo setting `ignore`, `skipTheseDirectories`, `proseBudget.exempt`, or cspell `ignorePaths` fails. A check whose tool has its own ignore file fails that file too, as its README says.

## Front matter

Front matter is optional. No check requires it, and the changelog and release tools read files with or without it. The [strip-front-matter skill](../skills/strip-front-matter/SKILL.md) removes a repo's fitness-only front matter.

## Shared configs

The opinionated tool configs are embedded inside the check binaries ([go/internal/sharedconf](../go/internal/sharedconf)). They materialize to a cache directory on demand. When a check needs a config, the repo's own config file wins.

Next comes an installed copy of the shared config package, and the embedded copy is the final fallback. A config may name plugins the consumer repo must install, which is the check's peer-tool contract.
