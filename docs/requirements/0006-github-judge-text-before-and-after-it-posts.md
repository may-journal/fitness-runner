# 0006 Judge Text Before and After It Posts

## Why

A comment or body that breaks the rules is caught before it posts. On GitHub, each one shows one verdict saying what to fix.

## Measurement

posted comments and bodies that meet the rules

-

posted comments and bodies

## Requirements

1. Judges a draft text
   1. GIVEN Developer has a draft text
      1. WHEN I run fitness with `--blob` or `comment-check` on it
         1. THEN the checks for that text judge it
         2. THEN the exit code says whether it passes
2. Denies a red post from Claude
   1. GIVEN Developer has Claude Code drafting a post in a fitness repo
      1. WHEN Claude is about to post it
         1. THEN a red comment or body is denied with its findings
3. Replies once to a broken comment
   1. GIVEN Developer has a comment that breaks the rules on GitHub
      1. WHEN comment-check judges it
         1. THEN it replies once saying what to fix
         2. THEN it updates the reply once fixed
4. Shows one Plan verdict
   1. GIVEN Developer has a Plan
      1. WHEN plan-check judges it
         1. THEN the Plan shows one current verdict
         2. THEN the Plan shows one current label
5. Shows one issue verdict
   1. GIVEN Developer has a labeled issue of any kind
      1. WHEN issue-check judges it
         1. THEN the issue shows one current verdict against its kind's template
6. Passes a ready pull request
   1. GIVEN Developer has a pull request
      1. WHEN pr-check judges it
         1. THEN it passes only if the title is ready
         2. THEN it passes only if the body is ready
         3. THEN it passes only if the closed issues are ready
7. Passes one new changelog entry
   1. GIVEN Developer has a pull request
      1. WHEN pr-check judges its changelog entry
         1. THEN it passes only one new entry file that the body links
