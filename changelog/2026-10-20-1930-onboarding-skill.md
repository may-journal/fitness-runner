---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.20.1930

- Feat: an onboarding skill teaches a new repo or agent to install fitness, run the checks, read failures, and follow the Plan flow.
- Feat: the skill sits beside the installer in `go/cmd/fitness-install/skills/onboarding/`, so its relatedConfigurations names the root `.fitnessrc.json`.
- Test: requirements 0099.1 and 0099.2 link the skill with the real installer and follow each doc link in it.
- Docs: the skill links the install, hook, workflow, check, and Plan docs and copies none of their rules.
- Fix: the skill orders the policy before install and links check docs by URL, and the link test skips fenced code.
