# 0143 Claude Code session hooks

## Why

A subagent fixes its own red work before it hands back, while it still has the context. A session that starts, resumes or compacts begins knowing what is red. Each session leaves numbers that show what the hooks caught.

## Measurement

subagent stops and session starts that hand Claude the findings on red work
-
subagent stops and session starts with red work

## Requirements

- 0143.1
    - Given Developer has a fitness repo with uncommitted files that fail a check
        - When Claude Code runs `fitness claude-hook` as a subagent stops, then again with the stop hook active
            - Then the first blocks the subagent with the findings and the second lets it stop
- 0143.2
    - Given Developer has a fitness repo
        - When Claude Code runs `fitness claude-hook` as a session starts, resumes or compacts
            - Then its best run adds the findings on uncommitted files to Claude's context within one second, and passing files add nothing
- 0143.3
    - Given Developer has a fitness repo and a Claude session that edited files
        - When Claude Code runs `fitness claude-hook` after each edit and as the session ends
            - Then the fitness cache folder logs a line per judged edit and the session's total per event, and the repo holds no log
