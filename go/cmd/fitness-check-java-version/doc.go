// cspell:ignore sdkmanrc jdks sdkman jabba libexec foojay

// Command fitness-check-java-version checks a Java repo pins its JDK and
// that the machine's JDKs satisfy the pin, through the Java language module.
// It is on by default.
//
// # Pins
//
//   - Version files at the repo root, the ones `actions/setup-java` reads: `.java-version`, `.sdkmanrc` (`java=`) and `.tool-versions` (`java`).
//   - Maven's `maven.compiler.release` or compiler `release`, one `${property}` deep.
//   - A Gradle toolchain's `JavaLanguageVersion.of(N)`.
//
// A pom that names no release takes one from its parents outside the repo,
// read from the local Maven repository. That is the one `.mvn/maven.config`
// or `MAVEN_OPTS` sets, else the user settings' `localRepository`, else
// `~/.m2/repository`. The properties of the pom and its parents in the repo
// come first, so a parent may map a property the pom sets. A parent the local
// repository lacks fails, when nothing else pins a release, asking to run the
// build once.
//
// # Toolchains
//
// A Gradle toolchain asks for its exact release, not a newer one. Gradle
// always finds the JDK it runs on, so that release passes at once. Any other
// release must be installed, and the check looks for it without starting
// Gradle.
//
// The lookup reads `JAVA_HOME`, macOS's `/usr/libexec/java_home -V`, and
// Gradle's detection folders. Those are `~/.gradle/jdks`, SDKMAN, asdf,
// jabba, `~/.jdks`, Homebrew, `/usr/lib/jvm` and the macOS JDK folders, where
// a bundle keeps `release` under `Contents/Home`. It adds the folders
// `org.gradle.java.installations.paths` names, and the variables
// `org.gradle.java.installations.fromEnv` names. Both split on commas, and
// come from the build root's `gradle.properties`, overridden by the Gradle
// user home's.
//
// A toolchain that sets `vendor` or J9 needs an installed JDK from it, else
// the check fails naming the JDK to install. A resolver exempts it, since
// Gradle then downloads the JDK: a download repository in
// `toolchainManagement`, or the Foojay plugin. Either may sit in the settings
// script or a user home init script. Commented-out code does not count, and
// `org.gradle.java.installations.auto-download=false` turns the exemption
// off.
//
// # Behavior
//
//   - Pass: the running JDK's feature release is at least each pinned one, since a newer JDK builds for an older release.
//   - Fail: no pin anywhere, with the snippets that add one.
//   - Fail: a pin above the running JDK, naming the file and the release it needs.
//   - Fail: a pin that names no release.
//   - The running JDK is `JAVA_HOME`'s, as the build tools pick it, else `java` on `PATH`.
//   - With no running JDK, the check fails with an install hint.
//   - No Maven or Gradle build in the repo: passes clean with `filesChecked: 0`.
package check
