---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.1545

- Fix: the shared Prettier config keeps `.github/ISSUE_TEMPLATE/**/*.md` and `.github/fitness-setup.md` at Prettier defaults.
- Fix: those files arrive from may-journal/.github in Prettier's default style, so sync PRs now pass `prettier`.
- Fix: the override's globs start at the repo root, since Prettier matches them from the cached config's folder.
