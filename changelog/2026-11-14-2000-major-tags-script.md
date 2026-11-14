### 2026.11.14.2000

> The major tag step of the reusable release workflow runs a checked-in script instead of inline shell.

- Refactor: `.github/actions/move-major-tags` holds the script and its action, and the workflow step calls the action.
- Test: requirements 0092.4 and 0092.6 to 0092.8 run the script file, with no change to what they prove.
- Docs: requirement 0092 says the major tag step runs that script.
- Docs: the script keeps the note on why only a missing tag is created.
