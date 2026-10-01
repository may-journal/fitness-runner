---
relatedConfigurations: ['../.fitnessrc.json']
---

# Adoption

Every may-journal repo runs the same checks by calling reusable workflows, which keeps repos current as checks grow.

## Reusable workflows

fitness-runner publishes [ci-reusable.yml](../.github/workflows/ci-reusable.yml), [pr-check-reusable.yml](../.github/workflows/pr-check-reusable.yml), [plan-check-reusable.yml](../.github/workflows/plan-check-reusable.yml), and [close-check-reusable.yml](../.github/workflows/close-check-reusable.yml). Each installs the checks with `go install`, then runs one `fitness` step. The install sets `GOPROXY=direct`, since the Go proxy can serve a stale `@main` after a merge. A consumer repo adds thin callers.

`.github/workflows/pr-check.yml`:

```yaml
name: PR check
on:
  pull_request:
    types: [opened, edited, reopened, synchronize]
  workflow_dispatch:
jobs:
  pr-check:
    uses: may-journal/fitness-runner/.github/workflows/pr-check-reusable.yml@main
    permissions:
      contents: read
      pull-requests: read
      issues: read
```

`.github/workflows/plan-check.yml`:

```yaml
name: Plan check
on:
  issues:
    types: [opened, edited, reopened, labeled]
  workflow_dispatch:
jobs:
  plan-check:
    uses: may-journal/fitness-runner/.github/workflows/plan-check-reusable.yml@main
    permissions:
      contents: read
      issues: write
```

## Closing issues

may-journal/.github syncs this [close-check](../.github/workflows/README.md#close-check) caller into each repo:

```yaml
name: Close check
on:
  issues:
    types: [closed]
jobs:
  close-check:
    uses: may-journal/fitness-runner/.github/workflows/close-check-reusable.yml@main
    permissions:
      contents: read
      issues: write
      pull-requests: read
```

It reopens issues closed with unchecked items; `fitness pr-check` fails PRs closing one.

## Reusable CI

Node repos get Node from `.nvmrc` and `npm ci` from the lockfile. SwiftLint is pinned by `swiftlint-version`; bump it with laptops.

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
    uses: may-journal/fitness-runner/.github/workflows/ci-reusable.yml@main
    with:
      swift: true
```

A pull request checks only the files it changes, and the push to `main` after merge checks every file. An org ruleset already runs this suite in every repo, so most repos need no caller.

## Version policy

The callers track `@main` to stay current. Pin a release tag when a repo needs a frozen check set, then bump on its own schedule.

## Per-repo pieces

Run `fitness init` to install the shared git hooks and point `core.hooksPath` at them. A repo runs every check, since each check skips itself when its language, tool, or config is absent. Go repos get `go vet`, `go test`, and `gofmt` this way. A `.fitnessrc.json` names only exceptions:

```json
{
  "disabledChecks": ["jscpd"],
  "ignore": ["profile/README.md"]
}
```

Repo-specific build or tests go in an executable `.githooks/pre-commit.local`.

## Org-wide

An org ruleset can run `pr-check-reusable.yml@main` in every repo with no caller file, since it carries a `pull_request` trigger. The org `.github` repo supplies the Plan issue template and conventions by default.

The org's `auto-merge.yml` runs as a ruleset workflow too, acting as the `may-journal-automation` App. It reads the org secrets `AUTOMATION_APP_ID` and `AUTOMATION_APP_KEY`.

A draft marked ready never starts it, so may-journal/.github syncs this caller into each repo as `.github/workflows/auto-merge-on-ready.yml`:

```yaml
name: Auto-merge on ready
on:
  pull_request_target:
    types: [ready_for_review]
permissions:
  contents: read
jobs:
  auto-merge:
    uses: may-journal/fitness-runner/.github/workflows/auto-merge.yml@main
    secrets: inherit
```
