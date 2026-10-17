---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.17.2307

- Fix: the Checkstyle config hint parses under Maven, and a failed Gradle build shows every cause line.
- Fix: a pom takes the release its parents map, and a Gradle toolchain must match an installed JDK without a resolver.
- Fix: spell and duplicate checks skip build wrapper files, and a duplicate failure names each file holding duplicated lines.
- Fix: a container path of `.` names the repo root, covering the product built there.
- Feat: the requirements check reads Java tests, so `test0001_1` owns 0001.1.
