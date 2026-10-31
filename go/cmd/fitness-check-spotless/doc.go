// Command fitness-check-spotless checks Java formatting with the [Spotless]
// plugin the repo's own build declares. The Java language module owns the
// build run and its judgment. It is on by default.
//
// # Behavior
//
//   - Runs once from each build root: `spotless:check` for Maven, `spotlessCheck` for Gradle.
//   - The [Java module] says what a build root is and which wrapper or tool runs it.
//   - Pass: Spotless finds nothing to rewrite.
//   - Fail: each file Spotless would rewrite, as `file is not formatted; run spotless:apply` (or `spotlessApply`).
//   - A failure that names no file reports the build's own error lines.
//   - No Maven or Gradle build in the repo: passes clean with `filesChecked: 0`.
//   - A Java repo with no running JDK fails with an install hint.
//   - A build that does not declare Spotless fails with the plugin snippet to add.
//
// # Formatter
//
// The build's own Spotless block picks the formatter, so the check never
// imposes a style. The hint suggests Google Java Format.
//
// [Spotless]: https://github.com/diffplug/spotless
// [Java module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/java/java.go
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
