---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1824

- Fix: Release approve approves as a machine user from `APPROVER_TOKEN`, since an App's approval does not count toward the required review.
- Fix: the run's own token reads the PR, its files and checks, so the approver's token needs only pull request access.
- Fix: without `APPROVER_TOKEN` the approval still skips with a notice, and a rejected token fails at once; the eligibility rules are unchanged.
- Docs: the workflows README names the secret and the read permissions each calling job grants.
- Test: requirements 0093 and 0094 cover the token check, the approver's login read from its token, and which token each step uses.
