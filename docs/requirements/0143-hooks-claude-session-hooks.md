# 0143 Claude Code session hooks

## Why

A subagent fixes its own red work before it hands back, while it still has the context. A session that starts, resumes or compacts begins knowing what is red. Each session leaves numbers that show what the hooks caught.

## Measurement

subagent stops and session starts that hand Claude the findings on red work
-
subagent stops and session starts with red work

## Requirements

- 0143.1
    - Given Developer has a fitness repo, and a subagent edited one of its uncommitted files that fail a check
        - When Claude Code runs `fitness claude-hook` as that subagent stops, again unmarked, then with the stop hook active
            - Then only the first blocks, on its file alone; another subagent blocks once too, and one that edited nothing passes
- 0143.2
    - Given Developer has a fitness repo on a feature branch off origin's default branch
        - When Claude Code runs `fitness claude-hook` as a session starts, resumes, clears or compacts
            - Then its best run adds findings on files the branch changed, committed or not, within one second; others add nothing
- 0143.3
    - Given Developer has a fitness repo and a Claude session that edited files
        - When Claude Code runs `fitness claude-hook` after each edit and as the session ends
            - Then the fitness cache folder logs a line per judged edit and the session's total per event, and the repo holds no log
- 0143.4
    - Given Developer has a fitness repo with red uncommitted files, on the default branch or a detached HEAD
        - When Claude Code runs `fitness claude-hook` as a session starts
            - Then it adds nothing to Claude's context
- 0143.5
    - Given Developer has a fitness repo with a failing Go test, and a subagent edited a Go file
        - When I run `fitness-install --claude-settings`, and Claude Code runs `fitness claude-hook` as the subagent stops
            - Then SubagentStop's timeout is 300 seconds, under Claude Code's ten-minute default, and no check over four minutes runs
- 0143.6
    - Given Developer has a fitness repo with uncommitted files that fail a check
        - When Claude Code runs `fitness claude-hook` as Claude stops twice in a session, never marked active
            - Then the first blocks, the second passes on the same findings, and new findings block again
