---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.13.2233

- Feat: changelog fails an entry file whose name lacks its heading's date, naming the expected `changelog/<yyyy-mm-dd-HHMM>-<change>.md` file.
- Feat: changelog-updated, pr-check and branch-check fail an added entry file with that bad name.
- Feat: the release fold orders entry files by name, newest first, and still deletes them.
- Docs: the changelog README, AGENTS.md, the PR template and requirements 0017, 0019 and 0069 show the dated name.
