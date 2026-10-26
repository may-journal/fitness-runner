# 0032 Checks Jscpd

## Why

I fix logic in one place, because copied blocks of code or docs never pile up unseen.

## Measurement

repos with duplicated lines at or under 1% of scanned lines
-
repos scanned

## Requirements

- 0032.1
    - Given Developer has two files sharing a block past 1% of all lines
        - When `jscpd` runs
            - Then it fails naming the share, the threshold, and each file holding duplicates
- 0032.12
    - Given Developer has two files sharing a block under 5 lines, and two sharing one under 50 tokens
        - When `jscpd` runs
            - Then it passes
- 0032.13
    - Given Developer stages a file sharing a block with a tracked file
        - When I commit
            - Then `jscpd` passes when their formats differ and fails when they match, reading only files of the staged formats
- 0032.9
    - Given Developer has a Markdown doc and a gitignored but still tracked JSON file sharing a block
        - When `jscpd` runs
            - Then it fails, since every tracked text file is compared
- 0032.11
    - Given Developer has copied lockfiles and build wrapper files, some marked `linguist-generated`
        - When `jscpd` runs
            - Then it passes, since all are left out
- 0032.6
    - Given Developer has binary files sharing a block
        - When `jscpd` runs
            - Then it passes, since binary files are skipped
- 0032.7
    - Given Developer has untracked files sharing a block
        - When `jscpd` runs
            - Then it passes, since only tracked files are scanned
- 0032.10
    - Given Developer has a doc template, a doc built from it, and a copy of that doc
        - When `jscpd` runs
            - Then it fails counting only the two docs, since a doc must repeat its template
