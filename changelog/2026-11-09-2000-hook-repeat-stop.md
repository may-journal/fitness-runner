### 2026.11.09.2000

- Fix: a repeat Stop event with the same input is judged again, as the hook no longer restamps its mark after judging.
- Test: requirement 0142.7 starts both copies of a hook event at once, as Claude Code does.
- Test: it installs once and shares one cache, and no longer depends on how long the first copy judges.
