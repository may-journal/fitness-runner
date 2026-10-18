---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1918

- Feat: `workflow-ref-exists` fails a may-journal `uses:` path missing at its tag or sha, naming the file, line, path, and ref.
- Feat: `workflow-ref-exists` skips with a note when GitHub cannot answer or the token cannot read a repo.
- Feat: the CI and fitness-suite workflows pass `GH_TOKEN` to the suite, so the check can ask GitHub.
