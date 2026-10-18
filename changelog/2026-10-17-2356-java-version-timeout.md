---
relatedConfigurations: ['../.fitnessrc.json']
---

### 2026.10.17.2356

- Fix: the `java-version` check no longer starts Gradle, so a cold daemon or a JDK download cannot slow it.
- Fix: a Gradle toolchain is matched against installed JDKs by release, vendor and implementation.
- Fix: a toolchain resolver exempts a release only when it registers a repository and auto-download is on.
- Docs: the README lists the JDK folders, `gradle.properties` rules and resolvers the lookup reads.
