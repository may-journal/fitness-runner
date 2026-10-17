---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# spotless

Checks Java formatting with the [Spotless](https://github.com/diffplug/spotless) plugin the repo's own build declares. On by default.

## Behavior

- Runs once from each build root: `spotless:check` for Maven, `spotlessCheck` for Gradle.
- The [Java module](../../internal/lang/java/java.go) says what a build root is and which wrapper or tool runs it.
- Pass: Spotless finds nothing to rewrite.
- Fail: each file Spotless would rewrite, as `file is not formatted; run spotless:apply` (or `spotlessApply`).
- A failure that names no file reports the build's own error lines.
- No Maven or Gradle build in the repo: passes clean with `filesChecked: 0`.
- A Java repo with no running JDK fails with an install hint.
- A build that does not declare Spotless fails with the plugin snippet to add.

## Config

The build's own Spotless block picks the formatter, so the check never imposes a style. The hint suggests Google Java Format.
