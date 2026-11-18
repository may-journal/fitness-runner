# 0005 Guard Commits and Pushes

## Why

A commit or push that breaks the bar stops on my machine, and Claude fixes red work before it hands back.

## Measurement

commits and pushes the hooks judged before review

-

commits and pushes

## Requirements

- 0005.1
  - Given Developer has a clone with no hooks or an old hooks path
    - When I run `fitness init` or commit
      - Then git runs the hooks the repo ships
- 0005.2
  - Given Developer has the hooks
    - When I commit
      - Then a change or message that breaks a check is refused with its errors
- 0005.3
  - Given Developer has a branch commit that links an issue
    - When I commit another link to it
      - Then the commit is refused, so each issue shows one link
- 0005.4
  - Given Developer has a feature branch with an open PR
    - When I push it
      - Then the push goes through only if its PR closes an approved Plan
- 0005.5
  - Given Developer has a branch with an open PR
    - When I run the suite or `branch-check`
      - Then its PR and Plan findings show before Actions runs
- 0005.6
  - Given Developer has branches whose PRs a push names
    - When I push them
      - Then each pushed branch's PR is checked once, and a bad one refuses the push
- 0005.7
  - Given Developer works with Claude Code in a fitness repo
    - When Claude edits, stops or starts a session
      - Then the hook shows the red checks Claude must fix
