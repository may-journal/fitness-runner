---
# How agents should work in this repo
relatedConfigurations: ['.fitnessrc.json']
---

# Agents

Guidance for AI agents working in this repository.

## Plans are Issues

Plans live as GitHub Issues under the `Plan` label, not as files in the repo. Each plan Issue follows the [Plan template](.github/ISSUE_TEMPLATE/plan.md): a one-line pitch, a `## Background`, and a `## What needs to happen` checklist.

A plan names the bug or feature it solves with a GitHub closing keyword (`Fixes #NN`) in its body. So the PR that closes the plan is required to close that issue too.

## Approval gate

An agent may work an Issue as it sees fit — implement it, branch, commit, and push. This is allowed once a human has approved it in a comment. Any clear approval counts: `I approve this plan`, `Approved`, or a comment containing `approve`. Until such a comment exists, do not start the work; refine the Issue description instead, then wait for approval.

## Once approved

Follow the repo's normal flow. Commit straight to `main`, run the full `fitness` suite before committing, and keep each change paired with a `CHANGELOG.md` entry.

Name the approved Issue with a `Plan #NN` trailer on the branch's first commit only. Each link adds to the Issue's timeline, and `issue-link-once` fails a repeat. The `pre-push` hook blocks a push that does not trace to an approved Plan Issue through the branch or its open PR. Chore and docs-only pushes are exempt.
