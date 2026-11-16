# 0175 Engine cache offline

## Why

Checks download on demand now, so old engine and check releases pile up in the cache, and nothing pruned them. With no network, a `"*"` pin found no checks at all. A check that needs the network failed on a read that could not work.

## Measurement

Cached releases a prune keeps that went unused for 30 days and no process uses
-
Cached releases

## Requirements

- 0175.1
    - Given Developer has a cache where some bundles, check releases and sets went unused for 31 days, the running engine's among them
        - When I run `fitness`, handed one of those check sets
            - Then it deletes only the unused ones no process uses, and nothing a link in the cache points to
- 0175.2
    - Given Developer has a cache pruned an hour ago, and one more check release then went unused for 31 days
        - When I run `fitness`, and again a day after the prune
            - Then the first run deletes nothing, and the second deletes that release
- 0175.3
    - Given Developer has a repo pinning a Markdown, a Go and a Swift check, and one with no fields, none cached
        - When I run `fitness prefetch`
            - Then it downloads and verifies every pinned check, so the next run downloads none
- 0175.4
    - Given Developer has a repo pinning `"*": "v1"`, every check cached by `fitness prefetch`, and a release index recorded 40 days ago
        - When I run `fitness --all` with GitHub silent
            - Then it says it is offline and matches against that index, and every cached check runs
- 0175.5
    - Given Developer has a repo pinning a check requiring `github`, one requiring `network`, and a Markdown check, none cached
        - When I run `fitness` with Markdown staged and GitHub's API host silent
            - Then the first two are skipped rows naming the requirement and offline, never fetched, and the Markdown check runs
- 0175.6
    - Given Developer has a check requiring `diff`
        - When I run `fitness --all`, then `fitness` with a file staged
            - Then the full run skips it, saying a full run has no diff, and the staged run runs it
