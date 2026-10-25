---
name: java-version
description: Fix a failing java-version check on the pinned JDK release. Use when a Java repo pins no JDK or the running JDK is older than a pin.
---

<!-- cspell:ignore temurin -->

# Pin and match the JDK release

A Java repo pins the release it builds with, and the running JDK is at least that release. [The check README](../../README.md) lists the files that count as a pin.

## Read the failure

- `no JDK version is pinned; add a .java-version file naming the release, such as 21, or set <maven.compiler.release>21</maven.compiler.release> in pom.xml, or java { toolchain { languageVersion = JavaLanguageVersion.of(21) } } in the Gradle build`: add one of the pins shown.
- `JDK 17 does not satisfy pom.xml (requires 21); install JDK 21 or point JAVA_HOME at one`: install the JDK the pin needs, or set `JAVA_HOME` to it.
- `build.gradle asks for a JDK 21 toolchain, which is not installed; install JDK 21, such as from https://adoptium.net, or configure a toolchain resolver to download it`: install that exact JDK, or configure a resolver.
- `.java-version pins JDK "latest", which names no release`: write a release number such as `21`.
- `Java not installed: install a JDK, such as Temurin from https://adoptium.net, or set JAVA_HOME`: install a JDK.

## Check it first

Run `fitness java-version` from the repo root. Exit 0 means a pin exists and the running JDK satisfies it.
