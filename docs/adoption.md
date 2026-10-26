---
relatedConfigurations: ['../.fitnessrc.json']
---
<!-- cspell:ignore temurin sdkmanrc -->

# Adoption

Every may-journal repo runs the same checks by calling reusable workflows, which keeps repos current as checks grow.

## Reusable workflows

fitness-runner publishes [ci-reusable.yml](../.github/workflows/ci-reusable.yml), [pr-check-reusable.yml](../.github/workflows/pr-check-reusable.yml), [issue-check-reusable.yml](../.github/workflows/issue-check-reusable.yml), [close-check-reusable.yml](../.github/workflows/close-check-reusable.yml), [dupe-check-reusable.yml](../.github/workflows/dupe-check-reusable.yml), and [comment-check-reusable.yml](../.github/workflows/comment-check-reusable.yml). Each uses a pinned binary release, then runs one `fitness` step. Consumers need no Go toolchain to install Fitness; see the [shell and CI guide](ci.md) for direct use. An explicit `ref` keeps the source-build path with `GOPROXY=direct`.

`.github/workflows/pr-check.yml`:

```yaml
name: PR check
on:
  pull_request:
    types: [opened, edited, reopened, synchronize]
  workflow_dispatch:
jobs:
  pr-check:
    uses: may-journal/fitness-runner/.github/workflows/pr-check-reusable.yml@v1
    permissions:
      contents: read
      pull-requests: read
      issues: read
```

`.github/workflows/issue-check.yml`:

```yaml
name: Issue check
on:
  issues:
    types: [opened, edited, reopened, labeled]
  workflow_dispatch:
jobs:
  issue-check:
    uses: may-journal/fitness-runner/.github/workflows/issue-check-reusable.yml@v1
    permissions:
      contents: read
      issues: write
```

## Issue kinds

Each `.github/ISSUE_TEMPLATE/*.md` is an issue kind, keyed by its `labels` front matter. [issue-check](../.github/workflows/README.md#issue-check) judges a labeled Issue by its template's sections and pitch, so a new kind is one template file. Plan and Epic add their own checks, and an Issue with no kind label passes. The older `plan-check-reusable.yml` still judges Plans only, for repos that call it.

## Closing issues

may-journal/.github syncs this [close-check](../.github/workflows/README.md#close-check) caller into each repo:

```yaml
name: Close check
on:
  issues:
    types: [closed]
jobs:
  close-check:
    uses: may-journal/fitness-runner/.github/workflows/close-check-reusable.yml@v1
    permissions:
      contents: read
      issues: write
      pull-requests: read
```

It reopens issues closed with unchecked items; `fitness pr-check` fails PRs closing one.

## Comments

may-journal/.github syncs this [comment-check](../.github/workflows/README.md#comment-check) caller into each repo. It judges every author but the known bots `fitness comment-check` lists, and carries no list of its own. A submitted review is judged in one job, so a review with ten comments bills one minute:

```yaml
name: Comment check
on:
  issue_comment:
    types: [created, edited]
  pull_request_review:
    types: [submitted]
  pull_request_review_comment:
    types: [edited]
  workflow_dispatch:
jobs:
  comment-check:
    uses: may-journal/fitness-runner/.github/workflows/comment-check-reusable.yml@v1
    permissions:
      contents: read
      issues: write
      pull-requests: write
```

Agents run `fitness comment-check --blob -` on a draft before posting, so a failing reply should be rare.

## Duplicate issues

The issue-check caller above also runs [dupe-check](../.github/workflows/README.md#dupe-check) in its one job, so a repo needs no dupe-check caller. A manual issue-check run also prints every pair's score, so a repo can tune its `dupeCheck` scores before turning the close tier on. `dupe-check-reusable.yml` stays for older callers and answers only a manual run.

## Reusable CI

Node repos get Node from `.nvmrc` and `npm ci` from the lockfile. Java repos get the JDK a root `.java-version`, `.sdkmanrc` or `.tool-versions` pins through `actions/setup-java`, from the vendor `java-distribution` names, `temurin` by default. SwiftLint is pinned by `swiftlint-version`; bump it with laptops.

`.github/workflows/ci.yml` (a Swift repo passes `swift: true`, which installs the pinned SwiftLint):

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
  workflow_dispatch:
jobs:
  fitness:
    uses: may-journal/fitness-runner/.github/workflows/ci-reusable.yml@v1
    with:
      swift: true
```

A pull request checks only the files it changes, and the push to `main` after merge checks every file. An org ruleset already runs this suite in every repo, so most repos need no caller.

## Version policy

The callers pin `@v1`, which each verified 1.x release moves, so repos get fixes with no bump. A new major version needs `@v2`.

## Per-repo pieces

Run `fitness init` to install the shared git hooks and point `core.hooksPath` at them. A repo runs every check, since each check skips itself when its language, tool, or config is absent. Go repos get `go vet`, `go test`, and `gofmt` this way. A `.fitnessrc.json` names only exceptions:

```json
{
  "repeatedStringLiterals": { "allow": ["dist"] }
}
```

Repo-specific build or tests go in an executable `.githooks/pre-commit.local`. It gets the budget of a check that declares none, so `timeoutMs` tunes it. Past that budget its process group is stopped and the commit is refused. The hooks also give each git call they make ten seconds.

Offline or behind a proxy, the hooks honor `HTTPS_PROXY` and `NO_PROXY`, as `gh` and git do. When the GitHub API host does not answer a two-second probe, network checks and the push gate report info. The [CI guide](ci.md#network-waits) says more.

## Org-wide

An org ruleset runs `fitness-suite.yml` from the `v1` tag in every repo with no caller file. Its one job runs the suite and `fitness pr-check`, so the org needs no separate pr-check ruleset. `pr-check-reusable.yml` stays for a repo that wants pr-check alone. The org `.github` repo supplies the Plan issue template and conventions by default.

The org's `auto-merge.yml` runs as a ruleset workflow too, acting as the `may-journal-automation` App. It reads the org secrets `AUTOMATION_APP_ID` and `AUTOMATION_APP_KEY`.

A draft marked ready or an approval never starts it, so may-journal/.github syncs this caller into each repo as `.github/workflows/auto-merge-on-ready.yml`:

```yaml
name: Auto-merge on ready
on:
  pull_request_target:
    types: [ready_for_review]
  pull_request_review:
    types: [submitted]
permissions:
  contents: read
jobs:
  auto-merge:
    if: github.event_name != 'pull_request_review' || (github.event.review.state == 'approved' && github.event.pull_request.head.repo.full_name == github.repository)
    uses: may-journal/fitness-runner/.github/workflows/auto-merge.yml@v1
    secrets: inherit
```
