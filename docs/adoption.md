---
relatedConfigurations: ['../.fitnessrc.json']
---

# Adoption

Every may-journal repo runs the same checks by calling reusable workflows, not by copying them. This keeps repos current as the check set grows.

## Reusable workflows

fitness-runner publishes two `workflow_call` workflows: [pr-check-reusable.yml](../.github/workflows/pr-check-reusable.yml) and [plan-check-reusable.yml](../.github/workflows/plan-check-reusable.yml). Each installs the checks with `go install`, then validates the calling repo's PR or Plan bodies. A consumer repo adds two thin callers.

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

## Version policy

The callers track `@main` to stay current. Pin a release tag when a repo needs a frozen check set, then bump on its own schedule.

## Per-repo pieces

Two bits stay local: a `.fitnessrc.json` tuned to the repo's languages, and the git hooks (`core.hooksPath`). The planned `fitness init` command writes both in one step. Until then, copy them from an adopted repo and adjust the check list.

## Org-wide

An org-level required workflow can run the reusable `pr-check` across repos with no caller file at all. Point the ruleset at `pr-check-reusable.yml@main`; it carries a `pull_request` trigger so a required workflow can run it. The org `.github` repo supplies the Plan issue template and conventions by default.
