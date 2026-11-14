### 2026.11.14.1900

- Fix: an installed `fitness-check-local-branch` no longer replaces the engine's judgment, since the program cannot judge without it.
- Test: requirement 0162 runs `local-branch` beside an installed copy that would fail, and the engine's judgment answers.
- Docs: the installed program only declares the check, so fitness-checks can ship it as a describe-only copy.
