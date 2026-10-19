---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.2200

- Fix: Auto-merge passes with a notice when GitHub still counts a pull request on the default branch as stacked and refuses auto-merge.
- Fix: any other auto-merge failure still fails the job.
- Test: requirements 0091.3 and 0091.4 prove both cases.
- Docs: the workflows README explains stacked pull requests GitHub moved onto the default branch.
