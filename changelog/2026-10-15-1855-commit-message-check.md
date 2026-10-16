---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.15.1855

- Fix: the pre-commit hook skips `semantic-commit` beside a commit-msg hook, and alone judges earlier branch commits, never the parent at HEAD.
- Fix: a full run has `semantic-commit` judge each of the branch's own commits, never inherited history, whatever the default branch is called.
- Test: requirements 0056.14 to 0056.17 prove each case with real git commits and the installer hook; 0056.9 and 0056.10 merge older acceptances.
