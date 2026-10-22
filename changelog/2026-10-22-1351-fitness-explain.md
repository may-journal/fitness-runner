### 2026.10.22.1351

- Feat: `fitness explain <check>` prints a check's scope, budget, and documentation from the installed binary.
- Feat: `go generate` renders each check's package doc comment and the config keys it reads into the binary as Markdown.
- Feat: a config struct field tagged `check:"<name>"` documents that key under the check; a key without a doc comment fails the generated check.
- Feat: an unknown name exits 2 and suggests the closest check names.
- Test: requirement 0111 explains every listed check, reads one as JSON, and mistypes a name.
