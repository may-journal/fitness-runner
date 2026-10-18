// cspell:ignore jdks
package requirements

import (
	"path/filepath"
	"strings"
	"testing"
)

// javaVersionRun runs `fitness-install -- java-version` on a Java repo with
// files written over it and a JDK 21 running.
func javaVersionRun(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	_, out, code := javaRun(t, javaRepo(t, files), "java-version", nil)
	return out, code
}

// javaVersionRelease is a Maven build whose compiler release is release.
func javaVersionRelease(release string) string {
	return "<project><properties><maven.compiler.release>" + release + "</maven.compiler.release></properties></project>\n"
}

func Test0080_1(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": "<project/>\n"})
	sees(t, out, code, 1, "no JDK version is pinned; add a .java-version file naming the release",
		"JavaLanguageVersion.of(21)")
}

func Test0080_2(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": "<project/>\n", ".java-version": "17\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0080_3(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": javaVersionRelease("25")})
	sees(t, out, code, 1, "JDK 21.0.2 does not satisfy pom.xml (requires 25); install JDK 25 or point JAVA_HOME at one")
}

func Test0080_4(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{
		"pom.xml": "<project><properties><java.version>25</java.version>" +
			"<maven.compiler.release>${java.version}</maven.compiler.release></properties></project>\n",
	})
	sees(t, out, code, 1, "does not satisfy pom.xml (requires 25)")
	// A parent from the local Maven repository maps the pom's property to
	// the compiler release, as framework parents do.
	home := t.TempDir()
	write(t, home, map[string]string{".m2/repository/" + javaVersionParentPom: javaVersionParent})
	child := "<project><parent><groupId>com.example</groupId><artifactId>parent</artifactId>" +
		"<version>1</version><relativePath/></parent><properties><java.version>25</java.version></properties></project>\n"
	javaVersionSees(t, javaRepo(t, map[string]string{"pom.xml": child}), home, "JDK 21.0.2 does not satisfy pom.xml (requires 25)")
	// The user's Maven settings may move the local repository.
	moved := t.TempDir()
	write(t, moved, map[string]string{".m2/settings.xml": "<settings><localRepository>${user.home}/maven</localRepository></settings>\n", "maven/" + javaVersionParentPom: javaVersionParent})
	javaVersionSees(t, javaRepo(t, map[string]string{"pom.xml": child}), moved, "does not satisfy pom.xml (requires 25)")
	// A pom outside the repo at the default relative path is no parent in
	// the repo, so the local repository's parent still counts.
	repo := javaRepo(t, map[string]string{"pom.xml": strings.Replace(child, "<relativePath/>", "", 1)})
	write(t, filepath.Dir(repo), map[string]string{"pom.xml": "<project><artifactId>parent</artifactId></project>\n"})
	javaVersionSees(t, repo, home, "does not satisfy pom.xml (requires 25)")
	// A module takes the property its parent in the repo sets.
	javaVersionSees(t, javaRepo(t, map[string]string{
		"pom.xml": "<project><artifactId>app</artifactId><modules><module>core</module></modules><properties><java.version>25</java.version></properties></project>\n",
		"core/pom.xml": "<project><parent><groupId>com.example</groupId><artifactId>app</artifactId><version>1</version></parent>" +
			"<properties><maven.compiler.release>${java.version}</maven.compiler.release></properties></project>\n",
	}), home, "does not satisfy core/pom.xml (requires 25)")
}

// javaVersionParentPom is where a local Maven repository keeps the parent
// pom com.example:parent:1.
const javaVersionParentPom = "com/example/parent/1/parent-1.pom"

// javaVersionParent maps java.version to the compiler release, as
// framework parents do.
const javaVersionParent = "<project><properties><java.version>17</java.version>" +
	"<maven.compiler.release>${java.version}</maven.compiler.release></properties></project>\n"

// javaVersionSees runs java-version on repo with home as the user's home
// and asserts it fails with want.
func javaVersionSees(t *testing.T, repo, home, want string) {
	t.Helper()
	rp := standIn(t, map[string][]response{"java": {javaJDK}})
	out, code := fitness(t, repo, append(rp.env, "JAVA_HOME=", "HOME="+home), "java-version")
	sees(t, out, code, 1, want)
}

func Test0080_5(t *testing.T) {
	t.Parallel()
	out, code := javaVersionRun(t, map[string]string{"pom.xml": "<project/>\n", ".java-version": "system\n"})
	sees(t, out, code, 1, `.java-version pins JDK "system", which names no release`)
}

func Test0080_6(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaVersionRelease("21")})
	out, code := fitness(t, repo, []string{pathWithout(t, "java"), "JAVA_HOME="}, "java-version")
	sees(t, out, code, 1, "Java not installed: install a JDK")
}

// javaVersionNoGradle is a Gradle stand-in that fails if it ever runs, since
// the check must find toolchain JDKs without starting Gradle.
var javaVersionNoGradle = response{Stderr: "gradle must not run\n", Exit: 1}

// javaVersionToolchain is a Gradle build whose toolchain asks for release.
func javaVersionToolchain(t *testing.T, release string, extra map[string]string) string {
	t.Helper()
	files := map[string]string{"build.gradle": "java { toolchain { languageVersion = JavaLanguageVersion.of(" + release + ") } }\n"}
	for name, body := range extra {
		files[name] = body
	}
	return javaRepo(t, files)
}

// javaVersionJDK is the release file of an installed JDK of release.
func javaVersionJDK(release string) string {
	return "JAVA_VERSION=\"" + release + ".0.1\"\n"
}

// javaVersionGradleRun runs java-version on repo with home as the user's
// home and a Gradle stand-in that fails if called, and returns what it
// printed, its exit code and the Gradle calls.
func javaVersionGradleRun(t *testing.T, repo, home string) (string, int, []string) {
	t.Helper()
	rp := standIn(t, map[string][]response{"java": {javaJDK}, "gradle": {javaVersionNoGradle}})
	out, code := fitness(t, repo, append(rp.env, "JAVA_HOME=", "GRADLE_USER_HOME=", "HOME="+home), "java-version")
	return out, code, rp.calls("gradle")
}

func Test0080_8(t *testing.T) {
	t.Parallel()
	// No installed JDK has the release and no resolver is configured. Gradle
	// never runs, so the check stays fast.
	out, code, calls := javaVersionGradleRun(t, javaVersionToolchain(t, "30", nil), t.TempDir())
	sees(t, out, code, 1, "build.gradle asks for a JDK 30 toolchain, which is not installed; install JDK 30")
	if len(calls) != 0 {
		t.Errorf("gradle calls = %q, want none", calls)
	}
	// A JDK in Gradle's own download folder counts as installed.
	home := t.TempDir()
	write(t, home, map[string]string{".gradle/jdks/jdk-30/release": javaVersionJDK("30")})
	out, code, calls = javaVersionGradleRun(t, javaVersionToolchain(t, "30", nil), home)
	sees(t, out, code, 0, "All 1 checks passed")
	if len(calls) != 0 {
		t.Errorf("gradle calls = %q, want none", calls)
	}
	// So does a folder gradle.properties names.
	jdks := t.TempDir()
	write(t, jdks, map[string]string{"release": javaVersionJDK("30")})
	props := map[string]string{"gradle.properties": "org.gradle.java.installations.paths=" + jdks + "\n"}
	out, code, _ = javaVersionGradleRun(t, javaVersionToolchain(t, "30", props), t.TempDir())
	sees(t, out, code, 0, "All 1 checks passed")
	// A toolchain resolver lets Gradle download the JDK.
	resolver := map[string]string{"settings.gradle": "toolchainManagement { jvm { } }\n"}
	out, code, _ = javaVersionGradleRun(t, javaVersionToolchain(t, "30", resolver), t.TempDir())
	sees(t, out, code, 0, "All 1 checks passed")
	// Gradle always finds the JDK it runs on.
	out, code, _ = javaVersionGradleRun(t, javaVersionToolchain(t, "21", nil), t.TempDir())
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0080_7(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "java-version", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "java-version")
}
