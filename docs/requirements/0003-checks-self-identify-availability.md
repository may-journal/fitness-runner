# 0003 Checks Self-Identify Availability

## Why

Every check decides for itself whether it can judge a repo. So any repo runs every check, new ones included, with nothing to configure.

## Measurement

checks that decide for themselves when they apply, adr-checks included
-
checks in the catalog

## Requirements

- 0003.1
    - Given Developer has a repo with no files or changes a check can judge, only fixtures, or none that opts the repo in
        - When the check runs
            - Then it scans nothing and passes with zero files
- 0003.2
    - Given Developer has a repo that chooses no checks
        - When fitness runs
            - Then the result rows list every installed check once, repo-wide checks first, then each language's, each group in name order
- 0003.3
    - Given Developer has a file that a file-reading check cannot read
        - When fitness runs that check
            - Then it skips that file, names it as info, and still judges the other files
- 0003.4
    - Given Developer has a check that asks a service, and the service cannot answer
        - When fitness runs that check
            - Then it passes with an info note naming what it could not judge and why
- 0003.5
    - Given Developer has a file that no-eslint-disable, repeated-string-literals, cspell, prose-budget, text-readability, or go-complexity cannot read
        - When fitness runs that check
            - Then it skips that file, names it as info, and still judges the other files
- 0003.6
    - Given Developer has a Go file that does not parse beside a file go-complexity cannot read
        - When fitness runs go-complexity
            - Then it fails on the parse error and names the unreadable file as info
- 0003.7
    - Given Developer has a requirement doc the requirements check cannot read, or git history it cannot replay
        - When fitness runs requirements
            - Then it names what it skipped as info and judges the rest
