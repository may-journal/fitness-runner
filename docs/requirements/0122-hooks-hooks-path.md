# 0122 Hooks Path

## Why

A clone set up before the repo's hook folder moved runs the shipped hooks after its next commit, with no command by hand.

## Measurement

clones whose stale hooks path is repaired by their next commit
-
clones with a stale hooks path

Source: `go/cmd/fitness/hooks_path.go:36`

## Requirements

- 0122.1
    - Given Developer has a clone whose hooks path is .githooks in a repo that ships .fitness-hooks
        - When I commit
            - Then git runs .fitness-hooks from then on
- 0122.2
    - Given Developer has a clone whose hooks path names a folder the repo does not ship
        - When I commit
            - Then git runs the folder the repo ships from then on
- 0122.3
    - Given Developer has a clone whose hooks path is outside the repo
        - When I commit
            - Then the path is kept, and the commit names it
- 0122.4
    - Given Developer has a clone whose hooks path still names .githooks
        - When I commit a message the hook refuses
            - Then one line names the old and new path before the refusal
- 0122.5
    - Given Developer has a clone with a stale hooks path in a repo that ships .fitness-hooks
        - When I run fitness init again
            - Then git runs .fitness-hooks from then on
