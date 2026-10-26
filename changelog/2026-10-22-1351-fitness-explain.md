### 2026.10.22.1351

- Feat: `fitness explain <check>` prints a check's scope, budget, and documentation from the installed binary.
- Feat: `go generate` renders each check's package doc comment and the config keys it reads into the binary as Markdown.
- Feat: a field tagged `check:"<name>"` documents that key and the keys nested in it, lists included; an undocumented key fails the generated check.
- Feat: an unknown name exits 2 and suggests the closest check names.
- Test: requirement 0111 explains every listed check, reads one as JSON, mistypes a name, and notes a failed handshake.
