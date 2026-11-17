# 0176 Engine documents with paths

## Why

Checks that read commit messages or GitHub bodies could not name a path, so they declared a scope or the blob flag. Each input now has a path, real or virtual, so one `files` pattern decides which checks it reaches. The old fields map onto that pattern.

## Measurement

Inputs a run judges that have a path its checks' patterns match
-
Inputs a run judges

## Requirements

- 0176.1
    - Given Developer has installed checks of scope `git`, of scope `blob`, of scope `blob` finding Markdown, and one excluding the folder
        - When I run all four scoped to a folder of Markdown
            - Then the `git` check and the Markdown one run, and the `blob` and excluding ones are skipped under no matching files
- 0176.2
    - Given Developer has installed checks of scope `blob`, of `blob: true` finding Go files, of Go files alone, and of scope `git` with `blob: true`
        - When I pipe a text to `fitness --blob`, then the same text with `--as x.go`
            - Then the first run judges it with the `blob` check, the second with the Go text check, each naming its skips
- 0176.3
    - Given Developer has a local check of scope `blob` and one of scope `github`
        - When I run `fitness list --json` and `fitness explain` on each
            - Then each shows the body paths' pattern for the first, and that the second requires `github`
- 0176.4
    - Given Developer has a repo pinning a Markdown check, a Go check, and a body check the branch's PR needs, none cached
        - When I run `local-branch` beside them with only Markdown staged
            - Then the run downloads the Markdown and body checks, and never the Go check
- 0176.5
    - Given Developer has a local check replacing a PR body check, whose `files` finds only comment bodies and which always fails
        - When I run `fitness pr-check` on a PR
            - Then the PR passes, since its body's path never reaches that check
