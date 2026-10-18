---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.17.2356

- Fix: the `java-version` check no longer starts Gradle, so a cold daemon or a JDK download cannot slow it.
- Fix: a Gradle toolchain release is found by looking for installed JDKs, and the check now ends within one minute.
- Docs: the README lists the JDK folders the lookup reads and the toolchain resolver that exempts a release.
