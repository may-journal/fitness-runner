---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.14.1938

- Fix: `jscpd` skips every file `doc-template` treats as a template, so a doc that repeats its template is no clone.
- Fix: docs built from one template are still compared with each other and with every other file.
- Refactor: `jscpd` and `doc-template` share one template detection, so the two checks never drift apart.
- Test: requirement 0032 proves a template is skipped while two copied docs still fail.
