### 2026.10.30.2129

- Feat: eslint, prettier, swiftlint, vitest-coverage-full, gofmt, go-vet, go-test, go-test-coverage and node-version report a missing tool as info.
- Feat: those checks report info, quoting the tool's last lines, when it could not start, was killed, or exited without a parsed finding.
- Fix: gofmt keeps its exit code and fails a syntax error as itself, never as an unformatted file on every output line.
- Feat: go-test-coverage reports environment errors as info, while config errors and coverage gaps still fail.
- Feat: dependency-currency adds an info note counting the packages it could not query from the registry.
