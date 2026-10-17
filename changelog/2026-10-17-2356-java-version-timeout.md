---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.17.2356

- Fix: the `java-version` check stops after 60 seconds instead of five minutes.
- Fix: a slow Gradle toolchain listing ends the check at the new limit.
- Docs: the check's source comment names the one-minute budget.
