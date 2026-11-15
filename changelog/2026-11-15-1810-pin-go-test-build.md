### 2026.11.15.1810

> The `go-test` requirement test builds the same fitness-checks every run.

- Test: requirement 0162.5 builds `go-test` from a pinned fitness-checks commit, not its moving main.
- Test: the test fails with a clear message when the suite names no released checks folder.
- Test: its inner Go test also fails when fitness cannot start.
