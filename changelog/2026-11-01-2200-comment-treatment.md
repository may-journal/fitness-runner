### 2026.11.01.2200

- Feat: every check declares how it treats code comments: judged, stripped or not applicable.
- Feat: `fitness explain`, `fitness list --json` and the docs/checks.md Comments column show that one declaration.
- Feat: `go generate` and the `generated` check fail a check package that declares no comment treatment.
- Test: real runs prove cspell judges comments and that repeated-string-literals, go-complexity and duplicate-code strip them.
