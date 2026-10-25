---
name: spotless
description: Fix a failing spotless check on Java formatting. Use when Spotless reports files that are not formatted.
---

<!-- cspell:ignore temurin -->

# Format Java with Spotless

Java files match the format the build's own Spotless block sets. [The check README](../../go/cmd/fitness-check-spotless/README.md) states the rules.

## Read the failure

- `src/main/java/App.java is not formatted; run spotless:apply`: run `spotless:apply` for Maven or `spotlessApply` for Gradle, then review the rewrite.
- `Spotless is not in the Maven build at <dir>; add it: ...`: add the plugin snippet shown. The Gradle wording names the Gradle build.
- `Spotless reported an error; run spotless:apply or the check by hand`: run the named command and read the build's own output.
- `Java not installed: install a JDK, such as Temurin from https://adoptium.net, or set JAVA_HOME`: install a JDK.

## Check it first

Run `fitness spotless` from the repo root. Exit 0 means Spotless has nothing to rewrite.
