// cspell:ignore mvnw diffplug

package requirements

import "testing"

// spotlessMaven declares Spotless in a Maven build.
const spotlessMaven = "<plugin><groupId>com.diffplug.spotless</groupId><artifactId>spotless-maven-plugin</artifactId></plugin>"

// spotlessClean is the build's answer when Spotless finds nothing to rewrite.
var spotlessClean = response{Match: []string{"spotless:check"}, Stdout: "[INFO] BUILD SUCCESS\n"}

// spotlessGradleClean is Gradle's answer for a clean spotlessCheck.
var spotlessGradleClean = response{Match: []string{"spotlessCheck"}, Stdout: "> Task :spotlessCheck\n\nBUILD SUCCESSFUL in 2s\n"}

// spotlessViolations is Maven's answer when Spotless would rewrite the
// Java source: the file under the violations header, then its diff.
var spotlessViolations = response{
	Match: []string{"spotless:check"},
	Stdout: "[INFO] Scanning for projects...\n" +
		"[ERROR] Failed to execute goal com.diffplug.spotless:spotless-maven-plugin:3.10.4:check (default-cli) on project app: The following files had format violations:\n" +
		"[ERROR]     " + javaSource + "\n" +
		"[ERROR]         @@ -1 +1 @@\n" +
		"[ERROR]         -class App {  }\n" +
		"[ERROR]         +class App {}\n" +
		"[ERROR] Run 'mvn spotless:apply' to fix these violations.\n",
	Exit: 1,
}

func Test0078_1(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaPom(spotlessMaven)})
	_, out, code := javaRun(t, repo, "spotless", map[string][]response{"mvn": {spotlessViolations}})
	sees(t, out, code, 1, javaSource+" is not formatted; run spotless:apply")
}

func Test0078_2(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{
		"pom.xml":                         javaPom(spotlessMaven),
		"core/pom.xml":                    javaPom(""),
		"core/src/main/java/Core.java":    "class Core {}\n",
		"core/src/test/resources/pom.xml": "<project/>\n",
		"web/pom.xml":                     javaPom(""),
		"web/src/main/java/web/Page.java": "class Page {}\n",
	})
	rp, out, code := javaRun(t, repo, "spotless", map[string][]response{"mvn": {spotlessClean}})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "spotless", "3")
	javaCalledOnce(t, rp, "mvn")
}

func Test0078_3(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaPom(spotlessMaven)})
	javaWrapper(t, repo, "mvnw")
	rp, out, code := javaRun(t, repo, "spotless", map[string][]response{"mvn": {spotlessClean}, "mvnw": {spotlessClean}})
	sees(t, out, code, 0, "All 1 checks passed")
	javaCalledOnce(t, rp, "mvnw")
	if calls := rp.calls("mvn"); len(calls) != 0 {
		t.Errorf("the build tool on PATH ran %q, want the wrapper only", calls)
	}
}

func Test0078_4(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"build.gradle": "plugins { id 'java'; id 'com.diffplug.spotless' version '8.10.4' }\n"})
	rp, out, code := javaRun(t, repo, "spotless", map[string][]response{"gradle": {spotlessGradleClean}})
	sees(t, out, code, 0, "All 1 checks passed")
	if !rp.called("gradle", "--no-daemon", "spotlessCheck") {
		t.Errorf("gradle calls = %q, want spotlessCheck with --no-daemon", rp.calls("gradle"))
	}
}

func Test0078_5(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaPom("")})
	_, out, code := javaRun(t, repo, "spotless", map[string][]response{"mvn": {spotlessClean}})
	sees(t, out, code, 1, "Spotless is not in the Maven build at .; add it:")
	javaSeesSnippet(t, out, "<artifactId>spotless-maven-plugin</artifactId>")
}

func Test0078_6(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "spotless", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "spotless")
}
