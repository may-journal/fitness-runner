---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.19.1530

- Perf: `java-test` runs Gradle with `--build-cache`, so unchanged tests are restored instead of rerun.
- Perf: a 60 s Gradle suite reruns in 4 s, and PetClinic's `java-test` in 13 s, down from 21 s.
- Perf: failed tests are never cached, and `org.gradle.caching=false` in `gradle.properties` turns the cache off.
