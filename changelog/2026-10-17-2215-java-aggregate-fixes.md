---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.17.2215

- Fix: an aggregate JaCoCo report covers a module only when it lists a module source by its declared package.
- Fix: a module with no product code keeps its own coverage report out of judgement once an aggregate exists.
- Fix: an aggregate report that does not parse is named alone, without every module report.
