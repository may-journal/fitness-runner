---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1126

- Fix: the `requirements` check no longer accepts an `Unpaired-Requirement` commit trailer; a changed requirement test or doc always fails without its partner.
- Fix: the `pre-commit` hook judges requirement pairs itself, so the `commit-msg` hook no longer runs the `requirements` check.
- Fix: a pull request no longer needs to list exemption trailers in its body, since none exist.
