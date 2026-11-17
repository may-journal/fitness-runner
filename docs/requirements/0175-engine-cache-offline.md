# 0175 Engine cache offline

## Why

Checks download on demand now, so old engine and check releases pile up in the cache, and nothing pruned them. With no network, a `"*"` pin found no checks at all. A check that needs the network failed on a read that could not work.

## Measurement

Cached releases a prune keeps that went unused for 30 days and no process uses
-
Cached releases

## Requirements

- 0175.1
    - Given Developer has a cache where bundles, releases, local builds and sets went unused for 31 days, some still used or linked
        - When I run `fitness` with GitHub answering, then after settings stop naming a bundle
            - Then it deletes only what nothing uses, links or names, and settles folders a prune set aside
- 0175.2
    - Given Developer has a cache pruned an hour ago, and one more check release then went unused for 31 days
        - When I run `fitness` a day after the prune offline, then online, then reading no index
            - Then only online runs delete it
- 0175.3
    - Given Developer has a repo pinning a Markdown, a Go and a Swift check, and one with no fields, none cached
        - When I run `fitness prefetch`
            - Then it downloads every pinned check, so the next run downloads none
- 0175.4
    - Given Developer has a repo pinning `"*": "v1"`, every check cached, and a release index recorded 40 days ago
        - When I run `fitness --all` with GitHub silent
            - Then it matches against that index, and every cached check runs
- 0175.5
    - Given Developer has a cached Markdown check, and uncached ones requiring `github`, `network` or nothing
        - When I run `fitness` with Markdown staged and GitHub's API host silent
            - Then the Markdown check runs, the others are never fetched, and each says it is offline
- 0175.6
    - Given Developer has a check requiring `diff`
        - When I run `fitness --all`, then `fitness` with a file staged
            - Then only the staged run runs it
- 0175.7
    - Given Developer has older engine and check versions beside newer cached ones
        - When I run `fitness` with GitHub answering
            - Then it deletes them, keeping exact pins, recent use, recently used sets' links, the running engine and one requested by version
