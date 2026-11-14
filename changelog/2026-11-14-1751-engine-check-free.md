### 2026.11.14.1751

- Refactor: commit-message Issue links are parsed in the public `gitx` package, which the engine and the issue-link-once check share.
- Refactor: the engine runs local-branch with its own PR and Plan judgments, so it imports no check package outside the generated built-ins map.
- Refactor: the local-branch check drops its exported `Judge` variable, which only the engine set.
- Test: requirement 0161 proves the engine's files, apart from `checks.go`, depend on no check package.
