---
relatedConfigurations: ['../../../.fitnessrc.json']
---

<!-- cspell:ignore sdkmanrc foojay -->

# java-version

Checks that a Java repo pins the JDK release it builds with, and that the running JDK satisfies every pin. On by default.

## Pins

- Version files at the repo root, the ones `actions/setup-java` reads: `.java-version`, `.sdkmanrc` (`java=`) and `.tool-versions` (`java`).
- Maven's `maven.compiler.release` or compiler `release`, one `${property}` deep.
- A Gradle toolchain's `JavaLanguageVersion.of(N)`.

A pom that names no release takes one from its parents outside the repo, read from the local Maven repository at `~/.m2/repository`. The pom's own properties come first, so a parent may map a property the pom sets. A parent the local repository lacks fails, when nothing else pins a release, asking to run the build once.

## Toolchains

A Gradle toolchain asks for its exact release, not a newer one. Unless the build configures a toolchain resolver, such as the foojay resolver plugin, the check runs Gradle's `javaToolchains` task from the build root. A release no listed JDK provides fails, naming the JDK to install.

## Behavior

- Pass: the running JDK's feature release is at least each pinned one, since a newer JDK builds for an older release.
- Fail: no pin anywhere, with the snippets that add one.
- Fail: a pin above the running JDK, naming the file and the release it needs.
- Fail: a pin that names no release.
- The running JDK is `JAVA_HOME`'s, as the build tools pick it, else `java` on `PATH`.
- With no running JDK, the check fails with an install hint.
- No Maven or Gradle build in the repo: passes clean with `filesChecked: 0`.
