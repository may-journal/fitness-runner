// cspell:ignore temurin

package requirements

import "testing"

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
	write(t, home, map[string]string{".m2/repository/com/example/parent/1/parent-1.pom": "<project><properties><java.version>17</java.version>" +
		"<maven.compiler.release>${java.version}</maven.compiler.release></properties></project>\n"})
	repo := javaRepo(t, map[string]string{"pom.xml": "<project><parent><groupId>com.example</groupId><artifactId>parent</artifactId>" +
		"<version>1</version><relativePath/></parent><properties><java.version>25</java.version></properties></project>\n"})
	rp := standIn(t, map[string][]response{"java": {javaJDK}})
	out, code = fitness(t, repo, append(rp.env, "JAVA_HOME=", "HOME="+home), "java-version")
	sees(t, out, code, 1, "JDK 21.0.2 does not satisfy pom.xml (requires 25)")
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

// javaVersionToolchains is Gradle's javaToolchains answer, saved from a
// real run, on a machine whose only JDK is 21.
var javaVersionToolchains = response{
	Match: []string{"javaToolchains"},
	Stdout: "> Task :javaToolchains\n\n + Options\n     | Auto-detection:     Enabled\n     | Auto-download:      Enabled\n\n" +
		" + Eclipse Temurin JDK 21 (21.0.12.1+1-LTS)\n     | Location:           /Library/Java/JavaVirtualMachines/temurin-21.jdk/Contents/Home\n" +
		"     | Language Version:   21\n     | Vendor:             Eclipse Temurin\n     | Architecture:       aarch64\n" +
		"     | Is JDK:             true\n     | Detected by:        Current JVM\n\nBUILD SUCCESSFUL in 4s\n",
}

func Test0080_8(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"build.gradle": "java { toolchain { languageVersion = JavaLanguageVersion.of(17) } }\n"})
	rp, out, code := javaRun(t, repo, "java-version", map[string][]response{"gradle": {javaVersionToolchains}})
	sees(t, out, code, 1, "build.gradle asks for a JDK 17 toolchain, which is not installed; install JDK 17")
	javaCalledOnce(t, rp, "gradle")
}

func Test0080_7(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "java-version", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "java-version")
}
