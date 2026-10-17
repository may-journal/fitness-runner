---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.17.2215

- Fix: a Maven aggregate JaCoCo report covers the modules it holds a group for, by artifact id.
- Fix: a Gradle aggregate covers a project when it lists one of its readable sources by declared package.
- Fix: once an aggregate exists, a module's own report that lists no class stays out of judgement.
- Fix: an aggregate report that does not parse is named alone, without every module report.
