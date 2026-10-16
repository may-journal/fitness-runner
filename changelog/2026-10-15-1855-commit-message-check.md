---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.15.1855

- Fix: the pre-commit hook skips `semantic-commit`, so a fresh branch's first commit no longer fails on its parent's subject.
- Fix: a full run has `semantic-commit` judge each of the branch's own commits, never history inherited from its base.
- Test: requirements 0056.11 to 0056.14 prove each case with real git commits, which name their own user and email; 0056.9 and 0056.10 merge older acceptances.
