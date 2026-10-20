---
name: onboarding
description: Adopt fitness in a repo or as an agent joining one. Use when a new repo needs the checks, a hook, or CI, when an agent asks how to install and run fitness, or when a fitness failure needs reading.
relatedConfigurations: ['../../.fitnessrc.json']
---

# Adopt fitness

Fitness runs every check this org holds code and issues to. Follow these steps in order, and read the linked doc at each step instead of guessing. Links point at the fitness-runner repo, since an installed skill has no docs beside it.

## Install and run

1. Commit the repo policy named in [Repo policy](https://github.com/may-journal/fitness-runner/blob/main/docs/ci.md#repo-policy).
2. Download the installer as [Download the Go installer](https://github.com/may-journal/fitness-runner/blob/main/docs/ci.md#download-the-go-installer) shows, then run the suite from the repo root with [Run checks](https://github.com/may-journal/fitness-runner/blob/main/docs/ci.md#run-checks).
3. Install the commit hook once with [Commit hook](https://github.com/may-journal/fitness-runner/blob/main/docs/ci.md#commit-hook). Never pass `--no-verify`.
4. Add the reusable workflows listed in [Adoption](https://github.com/may-journal/fitness-runner/blob/main/docs/adoption.md#reusable-workflows) so CI and issue checks run on every change.
5. Link the skills into your agent's skills folder with `fitness-install --link-skills <folder>`, as [Development](https://github.com/may-journal/fitness-runner/blob/main/docs/development.md#skills) explains.

## Read a failure

A failure names the check and the file. Open that check's README under [go/cmd](https://github.com/may-journal/fitness-runner/tree/main/go/cmd) as `fitness-check-<name>/README.md`, which states the rule and how to pass it. [Checks](https://github.com/may-journal/fitness-runner/blob/main/docs/checks.md) groups them by kind. Fix the code, never the check: no renaming, excluding or hiding to pass.

## Plans and pull requests

Plans are GitHub issues, and work starts only after a human approves one. The [Plan trailer check](https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-plan-trailer/README.md) explains the `Plan #NN` trailer, and the [changelog check](https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-changelog/README.md) explains the entry file. The [Plan template](https://github.com/may-journal/fitness-runner/blob/main/.github/ISSUE_TEMPLATE/plan.md) and the [PR template](https://github.com/may-journal/fitness-runner/blob/main/.github/PULL_REQUEST_TEMPLATE.md) show the sections the `plan-structure` and `pr-structure` checks require.

## When unsure

[Adoption](https://github.com/may-journal/fitness-runner/blob/main/docs/adoption.md) covers per-repo and org-wide pieces. A rule missing from the docs is a bug in the docs: file a Plan to fix them.
