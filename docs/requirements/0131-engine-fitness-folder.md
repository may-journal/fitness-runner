# 0131 Fitness Folder

## Why

My repo keeps each rule it writes in its own `fitness/<name>/` folder, and fitness runs, lists and explains it like a built-in check.

## Measurement

checks in a repo's fitness folder that run, list and explain like built-in checks
-
checks in a repo's fitness folder

## Requirements

- 0131.1
    - Given Developer has a shell check in `fitness/<name>/` with a README and no config
        - When the full suite runs
            - Then the check runs beside the bundled checks and reports its verdict
- 0131.2
    - Given Developer has a Go check in `fitness/<name>/` built on the check SDK
        - When fitness runs it twice
            - Then the first run builds it into the cache, the second reuses that build, and both report its verdict
- 0131.3
    - Given Developer has a Go check and no go on PATH
        - When the check runs
            - Then it reports red naming the check and where to get Go, and exits 1
- 0131.4
    - Given Developer has a check folder without a README, without a program, without `fitness/go.mod`, or with Go that does not build
        - When the suite starts
            - Then it reports red naming the folder and the fix, and exits 1
- 0131.5
    - Given Developer has checks in `fitness/` that find a problem
        - When `fitness check <name>` or `go test` in `fitness/` runs one
            - Then only that check judges the repo, and it fails naming the problem
- 0131.6
    - Given AI agent has a repo with a check in `fitness/`
        - When it runs `fitness list` and `fitness explain` on it
            - Then the list shows it with scope local and its README's first sentence, and explain prints the README
- 0131.7
    - Given Developer has a check in `fitness/` that declares a budget and runs past it
        - When the check runs
            - Then the list shows that budget, and the run stops the check at it and reports it red
- 0131.8
    - Given Developer has a check whose minFitness is newer than the installed fitness
        - When the check runs
            - Then it reports red naming the version to upgrade to, and exits 1
