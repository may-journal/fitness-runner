// cspell:ignore jacoco

// Command fitness-check-java-test runs Java tests once per build root and
// judges their JUnit XML results and [JaCoCo] coverage. The Java language
// module owns the build run and the judgment. It is on by default.
//
// # Behavior
//
//   - Maven runs `verify` past failing tests, so Surefire, Failsafe and the JaCoCo reports bound there all run.
//   - Gradle cleans and reruns `test` and `jacocoTestReport`, or `testCodeCoverageReport` under report aggregation. Another test task counts when one of those depends on it.
//   - Gradle adds `--build-cache`, so unchanged tests and reports are restored, not rerun. Failed tests are never cached.
//   - `org.gradle.caching=false` in `gradle.properties` turns the cache off.
//   - The [Java module] says what a build root is and which wrapper or tool runs it.
//   - No Maven or Gradle build in the repo: passes clean with `filesChecked: 0`.
//   - A Java repo with no running JDK fails with an install hint.
//   - A build that does not declare JaCoCo, or writes no XML report, fails with the plugin snippet to add.
//
// # Findings
//
//   - Each failing test, as `Class.method failed: message`, read from the JUnit XML the run wrote.
//   - A failed build with no failing test reports the build's own error lines.
//   - A build root with product code whose run ran no test. Test files that ran nothing get each detected framework's hint.
//   - A JaCoCo line or branch total below the shared goal, as `report: lines covered/total covered; the goal is 100%`.
//   - A report that does not parse, or a JaCoCo report that counts no lines.
//   - A Docker check that fails or hangs past 10 seconds, for a framework that needs Docker.
//
// # Reports
//
// Only reports the run itself wrote count. The JUnit XML comes from
// `target/surefire-reports`, `target/failsafe-reports` and Gradle's
// `build/test-results/<task>`.
//
// Coverage reads each report's own totals, never a sum of its classes. An
// aggregate report, Maven's `jacoco-aggregate` or Gradle's
// `testCodeCoverageReport`, replaces the module reports.
//
// # Frameworks
//
// The [frameworks] folder holds one commented file per framework, spotted
// from build files and imports. Each says how its tests reach the JUnit XML.
// A framework that needs a running service, as Testcontainers needs Docker,
// fails before the run when the service is down.
//
// [JaCoCo]: https://www.jacoco.org/jacoco/
// [Java module]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/java/java.go
// [frameworks]: https://github.com/may-journal/fitness-runner/blob/main/go/internal/lang/java/tests/frameworks
package check
