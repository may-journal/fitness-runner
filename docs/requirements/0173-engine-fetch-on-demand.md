# 0173 Engine fetch on demand

## Why

Every repo installed all 69 checks, a 142 MB download, though a commit reaches only a few of them. The release index copies each check's declared fields, so the engine fetches only the checks a run's files reach.

## Measurement

Checks a run downloads that its files reach
-
Checks a run downloads

## Requirements

- 0173.1
    - Given Developer has a repo pinning a Markdown, a Go and a Swift check, and one with no fields, none cached
        - When I run `fitness` with only Markdown, only Go, or only Swift files staged
            - Then each run downloads only the checks its files reach and the one with no fields, listing the rest as skipped
- 0173.2
    - Given Developer has the same repo
        - When I run `fitness --all`
            - Then the run downloads every pinned check
- 0173.3
    - Given Developer has the same repo, and the Markdown check's download does not match its index SHA-256
        - When I run `fitness` with only Markdown staged
            - Then its row is info naming the check and that it could not be fetched, never a pass
- 0173.4
    - Given Developer has the same repo, and the hooks' `FITNESS_WITH_CHECKS=pinned`
        - When I run `fitness-install` with only Markdown staged
            - Then it installs no check up front, and the engine downloads only the checks the Markdown reaches
