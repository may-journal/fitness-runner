# 0030 Checks Gofmt

## Why

I never review whitespace noise in Go diffs, because every Go file I commit is already formatted the standard way.

## Measurement

Go files `gofmt` leaves unchanged
-
Go files checked

## Requirements

- 0030.1
    - Given Developer has a committed Go file `gofmt` would rewrite
        - When fitness runs `gofmt`
            - Then it fails naming the file and the `gofmt -w` fix
- 0030.2
    - Given Developer has only formatted Go files
        - When fitness runs `gofmt`
            - Then it passes
- 0030.3
    - Given Developer has unformatted Go files under `testdata` or `vendor`
        - When fitness runs `gofmt`
            - Then it passes, since fixtures are not judged
- 0030.4
    - Given Developer has a repo with no Go files
        - When fitness runs `gofmt`
            - Then it passes having checked 0 files
- 0030.5
    - Given Developer has Go files and no Go toolchain on the path
        - When fitness runs `gofmt`
            - Then it fails with a hint to install Go
- 0030.6
    - Given Developer has an unformatted Go file that is not tracked by git
        - When fitness runs `gofmt`
            - Then it passes, since only tracked files are judged
