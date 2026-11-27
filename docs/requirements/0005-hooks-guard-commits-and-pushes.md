# 0005 Guard Commits and Pushes

## Why

A commit or push that breaks the bar stops on my machine, and Claude fixes red work before it hands back.

## Measurement

commits and pushes the hooks judged before review

-

commits and pushes

## Requirements

1. Installs the shipped hooks
   1. GIVEN Developer has a clone with no hooks or an old hooks path
      1. WHEN I run `fitness init` or commit
         1. THEN git runs the hooks the repo ships
2. Refuses a commit that breaks a check
   1. GIVEN Developer has the hooks
      1. WHEN I commit
         1. THEN a change or message that breaks a check is refused with its errors
3. Refuses a second issue link
   1. GIVEN Developer has a branch commit that links an issue
      1. WHEN I commit another link to it
         1. THEN the commit is refused
         2. THEN each issue shows one link
4. Gates a push on its Plan
   1. GIVEN Developer has a feature branch with an open PR
      1. WHEN I push it
         1. THEN the push goes through only if its PR closes an approved Plan
5. Shows PR findings early
   1. GIVEN Developer has a branch with an open PR
      1. WHEN I run the suite or `branch-check`
         1. THEN its PR findings show before Actions runs
         2. THEN its Plan findings show before Actions runs
6. Checks each pushed branch once
   1. GIVEN Developer has branches whose PRs a push names
      1. WHEN I push them
         1. THEN each pushed branch's PR is checked once
         2. THEN a bad one refuses the push
7. Shows Claude the red checks
   1. GIVEN Developer works with Claude Code in a fitness repo
      1. WHEN Claude edits or stops or starts a session
         1. THEN the hook shows the red checks Claude must fix
