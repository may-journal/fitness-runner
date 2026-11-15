# 0169 GitHub deleted branch install

## Why

A comment or review lands on a pull request after it merged and its branch went. The check should still judge it, so the install falls back to the merge commit.

## Measurement

source builds of a pull request ref that succeed or fail with a clear reason
-
source builds of a pull request ref

## Requirements

- 0169.1
    - Given Developer has a merged fitness-runner pull request, from a review or comment, whose head `go install` cannot fetch
        - When the source build runs on that head
            - Then it builds the merge commit instead, and says so in a notice
- 0169.2
    - Given Developer has an open pull request, another ref, another repo or a failed lookup, and `go install` cannot fetch the ref
        - When the source build runs on that ref
            - Then it fails with an error saying no other commit stands in, and builds nothing else
