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

func Test0080_7(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "java-version", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "java-version")
}
