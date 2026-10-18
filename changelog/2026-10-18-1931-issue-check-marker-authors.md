---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.18.1931

- Fix: issue-check and plan-check count a verdict marker only when the fitness bot wrote it, checked through the shared `BotLogin`.
- Fix: close-check counts its marker only in the fitness bot's comments, so a pasted marker cannot keep an issue closed.
- Docs: the workflows README says a pasted marker never counts, and `FITNESS_BOT_LOGIN` names another posting token.
