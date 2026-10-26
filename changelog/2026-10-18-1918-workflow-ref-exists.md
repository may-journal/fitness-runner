### 2026.10.18.1918

- Feat: `workflow-ref-exists` fails a may-journal `uses:` path missing at its tag or sha, naming the file, line, path, and ref.
- Feat: `workflow-ref-exists` keeps its findings and notes each call it could not verify, as when GitHub is unreachable.
- Feat: the CI and fitness-suite workflows pass `GH_TOKEN` to the suite, so the check can ask GitHub.
