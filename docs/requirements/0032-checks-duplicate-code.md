# 0032 Checks Duplicate Code

## Why

I fix logic in one place, because copied blocks of code or docs never pile up unseen.

## Measurement

repos with duplicated lines at or under 1% of scanned lines
-
repos scanned

## Requirements

- 0032.1
    - Given Developer has two files sharing a block past 1% of all lines
        - When `duplicate-code` runs
            - Then it fails naming the share, the threshold, and each file holding duplicates
- 0032.12
    - Given Developer has two files sharing a block under 5 lines, and two sharing one under 50 tokens
        - When `duplicate-code` runs
            - Then it passes
- 0032.13
    - Given Developer stages a file sharing a block with a tracked file
        - When I commit
            - Then `duplicate-code` passes when their formats differ and fails when they match, reading only files of the staged formats
- 0032.9
    - Given Developer has a Markdown doc and a gitignored but still tracked JSON file sharing a block
        - When `duplicate-code` runs
            - Then it fails, since every tracked text file is compared
- 0032.11
    - Given Developer has copied lockfiles and build wrapper files, some marked `linguist-generated`
        - When `duplicate-code` runs
            - Then it passes, since all are left out
- 0032.14
    - Given Developer has tracked binary files sharing a block, and untracked text files sharing one
        - When `duplicate-code` runs
            - Then it passes, since binary and untracked files are skipped
- 0032.10
    - Given Developer has a doc template, a doc built from it, and a copy of that doc
        - When `duplicate-code` runs
            - Then it fails counting only the two docs, since a doc must repeat its template
- 0032.15
    - Given Developer has two files sharing a block wrapped in `duplicate-code:ignore` markers, and two in `jscpd:ignore` markers
        - When `duplicate-code` runs
            - Then it passes, since both marker spellings leave the block out
