### 2026.11.09.1930

- Test: requirement 0142.7 starts both copies of a hook event at once, as Claude Code does.
- Test: it no longer waits for the first copy to finish, which took over 5s on a loaded runner.
- Test: the hook's duplicate window stays as it was, so a repeat Stop event is still judged.
