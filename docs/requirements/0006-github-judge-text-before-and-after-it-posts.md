# 0006 Judge Text Before and After It Posts

## Why

A comment or body that breaks the rules is caught before it posts. On GitHub, each one shows one verdict saying what to fix.

## Measurement

posted comments and bodies that meet the rules

-

posted comments and bodies

## Requirements

- 0006.1
  - Given Developer has a draft text
    - When I run fitness with `--blob` or `comment-check` on it
      - Then the checks for that text judge it, and the exit code says whether it passes
- 0006.2
  - Given Developer has Claude Code drafting a post in a fitness repo
    - When Claude is about to post it
      - Then a red comment or body is denied with its findings
- 0006.3
  - Given Developer has a comment that breaks the rules on GitHub
    - When comment-check judges it
      - Then it replies once saying what to fix, and updates the reply once fixed
- 0006.4
  - Given Developer has a Plan
    - When plan-check judges it
      - Then the Plan shows one current verdict and label
- 0006.5
  - Given Developer has a labeled issue of any kind
    - When issue-check judges it
      - Then the issue shows one current verdict against its kind's template
- 0006.6
  - Given Developer has a pull request
    - When pr-check judges it
      - Then it passes only when the title, body and closed issues are ready
- 0006.7
  - Given Developer has a pull request
    - When pr-check judges its changelog entry
      - Then it passes only one new entry file of its own that the body links
