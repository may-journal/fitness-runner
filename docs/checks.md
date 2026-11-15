# Checks

Each check is its own [fitness-checks](https://github.com/may-journal/fitness-checks) program, whose README states its rule. The engine runs only `local-branch` itself. The [fitness skill](../skills/fitness/SKILL.md) shows how to read any of them:

- `issue-check`, `pr-check`, `close-check`, and `comment-check` judge a GitHub blob with installed checks, then react; [`fitness --blob`](check-protocol.md#blob-input) judges any text
    - `issue-check` picks each issue's checks by its kind label and `.github/ISSUE_TEMPLATE`; `plan-check` is its Plan-only alias
    - `dupe-check` runs no check; it notes when a new issue's words repeat an older issue's
- Each [language module](../go/pkg/lang/langs/langs.go) claims its checks; one passes when its language is absent, and names the install when its tool is
- `pr-check` holds a PR to one [entry file](https://github.com/may-journal/fitness-checks/blob/main/cmd/fitness-check-changelog/README.md#entry-files), which the description links and never copies a bullet of

Org mode runs every check in the runner's order ([go/cmd/fitness/main.go](../go/cmd/fitness/main.go)). Each detects whether it applies. External mode also runs all checks by default, with an optional `checks` filter; see [CI setup](ci.md). A repo can add its own [local checks](check-protocol.md#local-checks), and an external repo can [disable checks](check-protocol.md#disabled-checks).

## List the checks

`fitness list` names every check this repo can run, installed, local or `local-branch`, in run order. Each row gives the check's summary, its scope, its stage, and whether the repo's config enables it. `fitness list --json` gives the same fields for an agent; `fitness check --list` prints the engine's own check.

A scope, declared in the check's `checkkit.Describe`, says what it reads: `files`, one given `blob` of text, `git` history, or live `github` metadata. The summary is the first sentence of the check's Go package doc comment, which `go generate` writes into the runner. The local [generated check](../fitness/generated/README.md) fails when that file is stale.

## Explain a check

`fitness explain <check>` prints the check's whole package doc comment, rendered as Markdown, and the `.fitnessrc.json` keys it reads. A config struct field tagged `check:"<name>"` names such a key, and its doc comment explains it. `--json` gives the name, summary, scope, stage, and body.

`fitness docs` lists the guide topics, and `fitness docs <topic>` prints one. A topic is a package whose doc comment carries a `//fitness:topic <name>` line, such as `protocol` for `pkg/checkkit`. It prints that package's doc comment and its exported declarations, so the guide is the code's own.

## Every tracked file

Checks judge every tracked file; only untracked files stay out, and outside git only `.git`. cspell and duplicate-code also skip files `.gitattributes` marks `linguist-generated`, such as lock files. A repo setting `ignore`, `skipTheseDirectories`, `proseBudget.exempt`, or cspell `ignorePaths` fails. A check whose tool has its own ignore file fails that file too, as its README says.

A Markdown check reads every ending GitHub renders as Markdown, plus MDX, as [mdx.Exts](../go/pkg/mdx/mdx.go) lists them. The mermaid checks also read `.mmd` and `.mermaid` files as one diagram each, and a fence inside a blockquote, since GitHub renders it.

## Front matter

A Markdown file needs no front matter. Never add `fitnessFunctions` or `relatedConfigurations`: no check reads them, and each check finds its files by path. `fitness strip-front-matter` deletes both keys where they remain, as the [strip-front-matter skill](../skills/strip-front-matter/SKILL.md) shows. Front matter another tool reads, such as an issue template's `name` and `labels`, stays.

## Shared configs

The opinionated tool configs are embedded inside the check binaries ([internal/sharedconf](https://github.com/may-journal/fitness-checks/blob/main/internal/sharedconf)). They materialize to a cache directory on demand. When a check needs a config, the repo's own config file wins.

Next comes an installed copy of the shared config package, and the embedded copy is the final fallback. A config may name plugins the consumer repo must install, which is the check's peer-tool contract.
