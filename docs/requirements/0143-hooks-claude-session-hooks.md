# 0143 Claude Code session hooks

## Why

A subagent fixes its own red work before it hands back, while it still has the context. A session that starts, resumes or compacts begins knowing what is red. Each session leaves numbers that show what the hooks caught.

## Measurement

subagent stops and session starts that hand Claude the findings on red work
-
subagent stops and session starts with red work

## Requirements

- 0143.1
    - Given Developer has a fitness repo, and a subagent edited an uncommitted file that fails a check
        - When Claude Code runs `fitness claude-hook` as that subagent stops, again unmarked, then marked active
            - Then only the first blocks, on its file alone; another subagent blocks once too; one that edited nothing passes
- 0143.2
    - Given Developer has a fitness repo on a feature branch
        - When Claude Code runs `fitness claude-hook` as a session starts, resumes, clears or compacts
            - Then its best run adds findings on files the branch changed, committed or not, within a second
- 0143.3
    - Given Developer has a fitness repo, a hook log with a long and an unreadable line, and edits
        - When Claude Code runs `fitness claude-hook` after each edit and as the session ends
            - Then the cache folder logs each edit's failed checks and finding lines, then the session's totals
- 0143.4
    - Given Developer has a fitness repo with red uncommitted files, on the default branch or detached
        - When Claude Code runs `fitness claude-hook` as a session starts
            - Then it adds nothing
- 0143.5
    - Given Developer has a fitness repo with a failing Go test, and a subagent edited a Go file
        - When I install the settings, and Claude Code runs `fitness claude-hook` as the subagent stops
            - Then SubagentStop's timeout is 300 seconds, and no check over four minutes runs
- 0143.6
    - Given Developer has a fitness repo with uncommitted files that fail a check
        - When Claude Code runs `fitness claude-hook` as Claude stops twice in a turn, unmarked, then next turn
            - Then the first stop of each turn blocks, and the second in a turn passes
- 0143.7
    - Given Developer has a fitness repo on a feature branch whose edit checks outlast the start hook's 15 second deadline
        - When Claude Code runs `fitness claude-hook` as a session starts
            - Then within 20 seconds it adds the findings so far and names the skipped checks
