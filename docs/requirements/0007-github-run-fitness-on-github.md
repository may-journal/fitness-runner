# 0007 Run Fitness on GitHub

## Why

My repo's Actions run fitness on every change, keep issues honest, and merge ready pull requests without a hand click. Each step that does real work calls a typed `fitness-ci` subcommand from the bundle, so a typo fails before a run (0008.7).

## Measurement

events whose workflow judged or acted on them

-

events

## Requirements

1. Judges changes before the full scan
   1. GIVEN Developer has a pull request
      1. WHEN fitness runs it in GitHub Actions
         1. THEN its changes are judged first
         2. THEN a full scan judges the rest
2. Installs checks on the right runner
   1. GIVEN Developer has the shared workflows
      1. WHEN a repo's Actions run fitness
         1. THEN they install its pinned checks on the right runner
         2. THEN they show a bill I can read
3. Keeps unfinished issues open
   1. GIVEN Developer has an issue with unfinished work
      1. WHEN it or a child closes
         1. THEN it stays open or comes back open
4. Closes a finished parent
   1. GIVEN Developer has a parent issue
      1. WHEN its last child closes as completed
         1. THEN the parent closes too
5. Points a repeat to the first issue
   1. GIVEN Developer has a new issue that repeats an older one
      1. WHEN dupe-check judges it
         1. THEN it points to the first issue once
6. Merges a ready pull request
   1. GIVEN Developer has a ready pull request or stack
      1. WHEN Auto-merge runs
         1. THEN it merges or a comment says why it waits
7. Approves only a lone dependency change
   1. GIVEN Developer has a green release or dev dependency pull request
      1. WHEN Release approve runs
         1. THEN it approves only one that changes nothing else
         2. THEN a file its base lacks reads as empty
8. Completes a Dependabot pull request
   1. GIVEN Developer has a Dependabot pull request
      1. WHEN dependabot-pr runs
         1. THEN it completes the title for pr-check
         2. THEN it completes the body for pr-check
         3. THEN it completes the changelog entry for pr-check
