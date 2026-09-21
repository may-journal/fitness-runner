---
# How agents should work in this repo
relatedConfigurations: ['.fitnessrc.json']
---

# Agents

Guidance for AI agents working in this repository.

## Plans are Issues

Plans live as GitHub Issues under the `Plan` label, not as files in the repo. Each plan Issue follows the [Plan template](.github/ISSUE_TEMPLATE/plan.md): a one-line pitch, a `## Background`, and a `## What needs to happen` checklist.

## Approval gate

An agent may work an Issue as it sees fit — implement it, branch, commit, and push — once a human has commented `I approve this plan` on that Issue. Until that comment exists, do not start the work. Refine the Issue description instead, then wait for approval.

## Once approved

Follow the repo's normal flow: commit straight to `main`, run the full `fitness` suite before committing, and keep each change paired with a `CHANGELOG.md` entry.
