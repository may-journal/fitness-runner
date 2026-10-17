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

// javaVersionNoJDK is Gradle's answer, saved from a real run, when it
// plans a compile whose toolchain asks for a JDK 17 it neither finds nor
// may download.
var javaVersionNoJDK = response{
	Match: []string{"compileJava", "--dry-run"},
	Stdout: "FAILURE: Build failed with an exception.\n\n* What went wrong:\n" +
		"Could not determine the dependencies of task ':compileJava'.\n" +
		"> Could not resolve all dependencies for configuration ':compileClasspath'.\n" +
		"   > Failed to calculate the value of task ':compileJava' property 'javaCompiler'.\n" +
		"      > Cannot find a Java installation on your machine (Mac OS X 27.0.1 aarch64) matching: {languageVersion=17, vendor=any vendor, implementation=vendor-specific, nativeImageCapable=false}. Toolchain download repositories have not been configured.\n" +
		"\n* Try:\n> Run with --stacktrace option to get the stack trace.\n\nBUILD FAILED in 4s\n",
	Exit: 1,
}

// javaVersionToolchain is a Gradle build whose toolchain asks for release.
func javaVersionToolchain(t *testing.T, release string) string {
	t.Helper()
	return javaRepo(t, map[string]string{"build.gradle": "java { toolchain { languageVersion = JavaLanguageVersion.of(" + release + ") } }\n"})
}

func Test0080_8(t *testing.T) {
	t.Parallel()
	rp, out, code := javaRun(t, javaVersionToolchain(t, "17"), "java-version", map[string][]response{"gradle": {javaVersionNoJDK}})
	sees(t, out, code, 1, "build.gradle asks for a JDK 17 toolchain, which is not installed; install JDK 17")
	javaCalledOnce(t, rp, "gradle")
	// A resolver lets Gradle download the JDK, so the plan succeeds even
	// above the running JDK.
	planned := response{Match: []string{"compileJava", "--dry-run"}, Stdout: ":compileJava SKIPPED\n\nBUILD SUCCESSFUL in 2s\n"}
	_, out, code = javaRun(t, javaVersionToolchain(t, "25"), "java-version", map[string][]response{"gradle": {planned}})
	sees(t, out, code, 0, "All 1 checks passed")
	// Gradle always finds the JDK it runs on, so it need not run.
	rp, out, code = javaRun(t, javaVersionToolchain(t, "21"), "java-version", map[string][]response{"gradle": {javaVersionNoJDK}})
	sees(t, out, code, 0, "All 1 checks passed")
	if calls := rp.calls("gradle"); len(calls) != 0 {
		t.Errorf("gradle calls = %q, want none for the running JDK's release", calls)
	}
}

func Test0080_7(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "java-version", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "java-version")
}
