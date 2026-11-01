# 0138 Pull Request Full Scan

## Why

Nothing merges while the whole repo fails. A pull request checks its own changes first, so I see my findings fast, then the full scan judges the rest.

## Measurement

pull requests merged while the full scan fails
-
pull requests merged

## Requirements

- 0138.1
    - Given Developer has a pull request whose changed files pass while a file it does not change fails
        - When fitness runs the pull request in GitHub Actions
            - Then stage 1 passes, stage 2 fails naming that file as outside the change, and the run exits 1
- 0138.2
    - Given Developer has a pull request in a repo whose every file passes
        - When fitness runs the pull request in GitHub Actions
            - Then both stages pass and the run exits 0
- 0138.3
    - Given Developer has a pull request in a repo whose config sets scopedPullRequests, and a file it does not change fails
        - When fitness runs the pull request in GitHub Actions
            - Then only stage 1 runs, it says a push to main scans every file, and the run exits 0
- 0138.4
    - Given Developer has a pull request whose changed file fails
        - When fitness runs the pull request in GitHub Actions
            - Then stage 1 fails with that file's errors, stage 2 does not run, and the run exits 1
- 0138.5
    - Given Developer has the hooks and a branch with a commit of its own
        - When I commit inside a pull request's GitHub Actions job
            - Then the pre-commit hook runs the checks once, in no stages, since the hooks never run the full scan
