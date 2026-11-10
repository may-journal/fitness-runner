# 0142 Claude Code hooks

## Why

Claude sees a failing check right after an edit or a draft and fixes it in the same turn. Nothing red reaches a commit, a push or a GitHub post, so no bot round trip is spent on it.

## Measurement

hook events that block red work before it is committed or posted
-
hook events on red work

## Requirements

- 0142.1
    - Given Developer has a fitness repo and Claude wrote a file that fails a check
        - When Claude Code runs `fitness claude-hook` after the edit
            - Then it blocks with that check's findings for that file alone, even when its name holds a glob
- 0142.2
    - Given Developer has a fitness repo with a failing Go test, and Claude edited a file
        - When Claude Code runs `fitness claude-hook` after the edit
            - Then no test-stage check runs, and its best run answers within 300 ms
- 0142.3
    - Given Developer has a fitness repo and Claude drafted a red PR body or comment
        - When Claude Code runs `fitness claude-hook` before the `gh` call posts it
            - Then it denies the call with the findings, wherever and however the shell passes the body
- 0142.4
    - Given Developer has a fitness repo
        - When Claude Code runs `fitness claude-hook` before a call that posts nothing
            - Then it exits 0 with no output
- 0142.5
    - Given Developer has a fitness repo with uncommitted files, any name, that fail a check
        - When Claude Code runs `fitness claude-hook` as Claude stops, then again with the stop hook active
            - Then the first blocks with the findings and the second lets Claude stop
- 0142.6
    - Given Developer has a repo with no fitness config and no fitness git hooks
        - When Claude Code runs `fitness claude-hook` after an edit that fails a check
            - Then it exits 0 with no output
- 0142.7
    - Given Developer has the hooks in both the fitness skill and the settings
        - When Claude Code runs `fitness claude-hook` twice for one event
            - Then only the first copy judges it, even when judging outlasts the 5s window
- 0142.8
    - Given Developer has a Claude Code settings file with settings and hooks of my own
        - When I run `fitness-install --claude-settings` with that file, twice
            - Then it holds each of the fitness skill's hooks once, run by the installed fitness, and mine stay
