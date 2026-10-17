---
relatedConfigurations: ['../../../.fitnessrc.json']
---

<!-- cspell:ignore sdkmanrc -->

# java-version

Checks that a Java repo pins the JDK release it builds with, and that the running JDK satisfies every pin. On by default.

## Pins

- Version files at the repo root, the ones `actions/setup-java` reads: `.java-version`, `.sdkmanrc` (`java=`) and `.tool-versions` (`java`).
- Maven's `maven.compiler.release` or compiler `release`, one `${property}` deep.
- A Gradle toolchain's `JavaLanguageVersion.of(N)`.

## Behavior

- Pass: the running JDK's feature release is at least each pinned one, since a newer JDK builds for an older release.
- Fail: no pin anywhere, with the snippets that add one.
- Fail: a pin above the running JDK, naming the file and the release it needs.
- Fail: a pin that names no release.
- The running JDK is `JAVA_HOME`'s, as the build tools pick it, else `java` on `PATH`.
- With no running JDK, the check fails with an install hint.
- No Maven or Gradle build in the repo: passes clean with `filesChecked: 0`.
