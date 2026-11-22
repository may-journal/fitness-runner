# 0007 Run Fitness on GitHub

## Why

My repo's Actions run fitness on every change, keep issues honest, and merge ready pull requests without a hand click. Each step that does real work calls a typed `fitness-ci` subcommand from the bundle, so a typo fails before a run (0008.7).

## Measurement

events whose workflow judged or acted on them

-

events

## Requirements

- 0007.1
  - Given Developer has a pull request
    - When fitness runs it in GitHub Actions
      - Then its changes are judged first, then a full scan judges the rest
- 0007.2
  - Given Developer has the shared workflows
    - When a repo's Actions run fitness
      - Then they install its pinned checks on the right runner, with a bill I can read
- 0007.3
  - Given Developer has an issue with unfinished work
    - When it or a child closes
      - Then it stays open or comes back open
- 0007.4
  - Given Developer has a parent issue
    - When its last child closes as completed
      - Then the parent closes too
- 0007.5
  - Given Developer has a new issue that repeats an older one
    - When dupe-check judges it
      - Then it points to the first issue, once
- 0007.6
  - Given Developer has a ready pull request or stack
    - When Auto-merge runs
      - Then it merges, or a comment says why it waits
- 0007.7
  - Given Developer has a green release or dev dependency pull request
    - When Release approve runs
      - Then it approves only one that changes nothing else
- 0007.8
  - Given Developer has a Dependabot pull request
    - When dependabot-pr runs
      - Then it completes the title, body and changelog entry for pr-check
