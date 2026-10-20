---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1700

- Feat: Release approve takes `kind: dependency` to approve a green `build(deps)` pull request that only bumps dev dependencies.
- Feat: a bump qualifies only when `package.json` and its lockfile change dev-only registry packages, each by a patch or minor version.
- Feat: runtime dependencies, major bumps and any other file stay with a person, with a notice saying why.
- Docs: the workflows README explains the dependency rule and how a repo calls it.
- Test: requirement 0094 covers the approved bumps, each refusal, and which check each kind runs.
