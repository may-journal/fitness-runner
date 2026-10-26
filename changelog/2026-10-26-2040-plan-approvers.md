---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.26.2040
- Fix: the pre-push gate counts only a comment whose first line starts with `I approve`, so "not approved" no longer passes a Plan.
- Feat: a new `planApprovers` setting lists the logins whose approval counts; unset, the repo's owner and org members may approve.
- Feat: the gate reads each comment's author and association in its one `gh issue view`, so it asks GitHub nothing more.
- Feat: a refused push names who may approve the Plan.
