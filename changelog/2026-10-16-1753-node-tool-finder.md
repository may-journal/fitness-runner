---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.16.1753

- Refactor: the Node checks share one tool finder and one JavaScript project rule, in `go/internal/nodejs`.
- Refactor: each check's script extension list is a named subset of one shared list, so results stay the same.
- Fix: `vitest-coverage-full` skips a `node_modules/.bin/vitest` without an execute bit and tries `PATH`, as eslint, prettier and Biome already did.
- Test: requirement 0060.7 proves the non-executable Vitest case.
