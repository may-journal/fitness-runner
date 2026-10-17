---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# checkstyle

Lints Java with the [Checkstyle](https://checkstyle.org) plugin the repo's own build declares, and holds its config to a complexity ceiling. On by default.

## Behavior

- Runs once from each build root, main and test sources alike. Maven runs `checkstyle:check` with warnings counted as violations.
- Gradle runs `checkstyleMain` and `checkstyleTest` with `--rerun`, since an up-to-date task prints no warnings.
- Pass: no finding, warnings included.
- Fail: each finding, as `file:line[:col] - message (Rule)`.
- A failure that names no finding reports the build's own error lines.
- No Maven or Gradle build in the repo: passes clean with `filesChecked: 0`.
- A Java repo with no running JDK fails with an install hint.
- A build that does not declare Checkstyle fails with the plugin snippet to add.

## Complexity ceiling

Every tracked Checkstyle config must hold the `CyclomaticComplexity` module, which caps each method's branches. Configs in a test source set's folder, such as `src/test/resources`, are test inputs and left out. A repo with no config fails asking for `config/checkstyle/checkstyle.xml`, with the document type line Maven's plugin needs to read it. The hints suggest a `max` of 5; the repo's config sets its own.

Maven runs with the repo's only config, or else with `config/checkstyle/checkstyle.xml`, unless the build names its own.
