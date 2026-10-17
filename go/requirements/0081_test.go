// cspell:ignore jacoco testcontainers karatelabs opentest sessioninfo Postgre

package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// javaTestMaven declares JaCoCo in a Maven build.
const javaTestMaven = "<plugin><groupId>org.jacoco</groupId><artifactId>jacoco-maven-plugin</artifactId></plugin>"

// javaTestGradle is a Gradle build that applies JaCoCo.
const javaTestGradle = "plugins { id 'java'; id 'jacoco' }\n"

// javaTestPassed is a JUnit XML report of one passing test.
const javaTestPassed = `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="AppTest" tests="1" failures="0" errors="0" skipped="0"><testcase name="adds" classname="example.AppTest" time="0.01"/></testsuite>
`

// javaTestFailed is a JUnit XML report of one passing and one failing test.
const javaTestFailed = `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="AppTest" tests="2" failures="1" errors="0" skipped="0">
<testcase name="adds" classname="example.AppTest" time="0.01"/>
<testcase name="subtracts" classname="example.AppTest" time="0.01"><failure message="expected: &lt;1&gt; but was: &lt;2&gt;" type="org.opentest4j.AssertionFailedError">org.opentest4j.AssertionFailedError: expected: &lt;1&gt; but was: &lt;2&gt;
	at example.AppTest.subtracts(AppTest.java:12)</failure></testcase>
</testsuite>
`

// javaTestCoverage is a JaCoCo XML report whose own totals hold covered
// and missed lines and cover every branch.
func javaTestCoverage(covered, missed string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><!DOCTYPE report PUBLIC "-//JACOCO//DTD Report 1.1//EN" "report.dtd">` +
		`<report name="app"><sessioninfo id="s" start="1" dump="2"/><package name="example"><counter type="LINE" missed="0" covered="1"/></package>` +
		`<counter type="LINE" missed="` + missed + `" covered="` + covered + `"/><counter type="BRANCH" missed="0" covered="2"/></report>
`
}

// javaTestFull is a JaCoCo report with every line and branch covered.
var javaTestFull = javaTestCoverage("5", "0")

// javaTestMavenRun is Maven's answer to verify: it writes files and passes,
// as it does past failing tests.
func javaTestMavenRun(files map[string]string) response {
	return response{Match: []string{"verify"}, Files: files, Stdout: "[INFO] BUILD SUCCESS\n"}
}

func Test0081_1(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaPom(javaTestMaven)})
	run := javaTestMavenRun(map[string]string{
		"target/surefire-reports/TEST-example.AppTest.xml": javaTestFailed,
		"target/site/jacoco/jacoco.xml":                    javaTestFull,
	})
	_, out, code := javaRun(t, repo, "java-test", map[string][]response{"mvn": {run}})
	sees(t, out, code, 1, "example.AppTest.subtracts failed: expected: <1> but was: <2>")
	if strings.Contains(read(out), "example.AppTest.adds") {
		t.Errorf("only the failing test may be named:\n%s", out)
	}
	// A report cut short names itself, never a missing test run.
	broken := javaTestMavenRun(map[string]string{"target/surefire-reports/TEST-example.AppTest.xml": "<testsuite><testcase"})
	_, out, code = javaRun(t, javaRepo(t, map[string]string{"pom.xml": javaPom(javaTestMaven)}), "java-test", map[string][]response{"mvn": {broken}})
	sees(t, out, code, 1, "target/surefire-reports/TEST-example.AppTest.xml is not a JUnit XML report; rerun the build")
}

func Test0081_2(t *testing.T) {
	t.Parallel()
	maven := javaTestMavenRun(map[string]string{
		"target/surefire-reports/TEST-example.AppTest.xml": javaTestPassed,
		"target/failsafe-reports/TEST-example.AppIT.xml":   javaTestPassed,
		"target/site/jacoco/jacoco.xml":                    javaTestFull,
	})
	rp, out, code := javaRun(t, javaRepo(t, map[string]string{"pom.xml": javaPom(javaTestMaven)}), "java-test", map[string][]response{"mvn": {maven}})
	sees(t, out, code, 0, "All 1 checks passed")
	javaCalledOnce(t, rp, "mvn")
	gradle := response{Match: []string{"test", "jacocoTestReport"}, Stdout: "BUILD SUCCESSFUL in 3s\n", Files: map[string]string{
		"build/test-results/test/TEST-example.AppTest.xml":          javaTestPassed,
		"build/test-results/integrationTest/TEST-example.AppIT.xml": javaTestPassed,
		"build/reports/jacoco/test/jacocoTestReport.xml":            javaTestFull,
	}}
	rp, out, code = javaRun(t, javaRepo(t, map[string]string{"build.gradle": javaTestGradle}), "java-test", map[string][]response{"gradle": {gradle}})
	sees(t, out, code, 0, "All 1 checks passed")
	javaCalledOnce(t, rp, "gradle")
	// Clean rules rerun the tasks on every Gradle version; --rerun needs 7.6.
	if !rp.called("gradle", "cleanTest test cleanJacocoTestReport jacocoTestReport") || rp.called("gradle", "--rerun") {
		t.Errorf("gradle calls = %q, want clean rules before each task and no --rerun", rp.calls("gradle"))
	}
}

func Test0081_3(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaPom(javaTestMaven)})
	run := javaTestMavenRun(map[string]string{
		"target/surefire-reports/TEST-example.AppTest.xml": javaTestPassed,
		"target/site/jacoco/jacoco.xml":                    javaTestCoverage("4", "1"),
	})
	_, out, code := javaRun(t, repo, "java-test", map[string][]response{"mvn": {run}})
	sees(t, out, code, 1, "target/site/jacoco/jacoco.xml: lines 4/5 covered; the goal is 100%")
	// A report without line counts can't show the goal is met.
	noLines := javaTestMavenRun(map[string]string{
		"target/surefire-reports/TEST-example.AppTest.xml": javaTestPassed,
		"target/site/jacoco/jacoco.xml":                    `<report name="app"><counter type="INSTRUCTION" missed="0" covered="9"/></report>`,
	})
	_, out, code = javaRun(t, javaRepo(t, map[string]string{"pom.xml": javaPom(javaTestMaven)}), "java-test", map[string][]response{"mvn": {noLines}})
	sees(t, out, code, 1, "target/site/jacoco/jacoco.xml counts no lines; compile with debug information")
}

func Test0081_4(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"pom.xml":                      javaParent(javaTestMaven, "core", "extra", "it", "kt", "coverage"),
		"core/pom.xml":                 javaTestModule("core"),
		"core/src/main/java/Core.java": "class Core {}\n",
		// Maven's aggregate covers the modules it holds a group for, so a
		// source another group lists leaves extra uncovered.
		"extra/pom.xml":                 javaTestModule("extra"),
		"extra/src/main/java/Core.java": "class Core {}\n",
		// A module of tests alone has no code of its own to cover.
		"it/pom.xml":                           javaTestModule("it"),
		"it/src/test/java/example/FlowIT.java": "class FlowIT {}\n",
		// A module of other languages has code the aggregate may miss.
		"kt/pom.xml":                         javaTestModule("kt"),
		"kt/src/main/kotlin/example/Util.kt": "package example\nobject Util\n",
		"coverage/pom.xml":                   javaTestModule("coverage"),
	}
	reports := map[string]string{
		"core/target/surefire-reports/TEST-example.CoreTest.xml": javaTestPassed,
		"core/target/site/jacoco/jacoco.xml":                     javaTestCoverage("2", "3"),
		"extra/target/site/jacoco/jacoco.xml":                    javaTestCoverage("3", "2"),
		"it/target/site/jacoco/jacoco.xml":                       `<report name="it"><counter type="INSTRUCTION" missed="0" covered="0"/></report>`,
		"kt/target/site/jacoco/jacoco.xml":                       javaTestCoverage("1", "4"),
		"coverage/target/site/jacoco-aggregate/jacoco.xml":       javaTestAggregate("4", "1", `<group name="core">`+javaTestListed("", "Core.java")+`</group>`),
	}
	_, out, code := javaRun(t, javaRepo(t, files), "java-test", map[string][]response{"mvn": {javaTestMavenRun(reports)}})
	sees(t, out, code, 1, "coverage/target/site/jacoco-aggregate/jacoco.xml: lines 4/5 covered", "extra/target/site/jacoco/jacoco.xml: lines 3/5 covered", "kt/target/site/jacoco/jacoco.xml: lines 1/5 covered")
	javaTestNotJudged(t, out, "core/target/site/jacoco/jacoco.xml", "it/target/site/jacoco/jacoco.xml")
	// An aggregate that does not parse covers what is unknown, so it is
	// named alone.
	reports["coverage/target/site/jacoco-aggregate/jacoco.xml"] = "<report"
	_, out, code = javaRun(t, javaRepo(t, files), "java-test", map[string][]response{"mvn": {javaTestMavenRun(reports)}})
	sees(t, out, code, 1, "coverage/target/site/jacoco-aggregate/jacoco.xml is not a JaCoCo XML report; rerun the build")
	javaTestNotJudged(t, out, "extra/target/site/jacoco/jacoco.xml")
	javaTestGradleAggregate(t)
}

// javaTestGradleAggregate proves Gradle's aggregate, which lists packages
// without module groups, covers a project when it lists one of its
// sources by declared package.
func javaTestGradleAggregate(t *testing.T) {
	t.Helper()
	files := map[string]string{
		"settings.gradle":              "include 'core', 'extra', 'locked'\n",
		"build.gradle":                 "plugins { id 'java'; id 'jacoco-report-aggregation' }\n",
		"core/build.gradle":            javaTestGradle,
		"core/src/main/java/Core.java": "class Core {}\n",
		// A class the aggregate's filters leave out keeps core covered.
		"core/src/main/java/example/Port.java": "package example;\ninterface Port {}\n",
		// The same file name in another package is another source.
		"extra/build.gradle":                    javaTestGradle,
		"extra/src/main/java/example/Core.java": "/* a; b */\npackage example;\nclass Core {}\n",
		// A source that cannot be read is not listed.
		"locked/build.gradle":                    javaTestGradle,
		"locked/src/main/java/example/Core.java": "package example;\nclass Core {}\n",
	}
	repo := javaRepo(t, files)
	mustDo(t, os.Chmod(filepath.Join(repo, "locked/src/main/java/example/Core.java"), 0))
	run := response{Match: []string{"testCodeCoverageReport"}, Stdout: "BUILD SUCCESSFUL in 3s\n", Files: map[string]string{
		"core/build/test-results/test/TEST-example.CoreTest.xml":                 javaTestPassed,
		"core/build/reports/jacoco/test/jacocoTestReport.xml":                    javaTestCoverage("2", "3"),
		"extra/build/reports/jacoco/test/jacocoTestReport.xml":                   javaTestCoverage("3", "2"),
		"locked/build/reports/jacoco/test/jacocoTestReport.xml":                  javaTestCoverage("1", "4"),
		"build/reports/jacoco/testCodeCoverageReport/testCodeCoverageReport.xml": javaTestAggregate("5", "0", javaTestListed("", "Core.java")),
	}}
	_, out, code := javaRun(t, repo, "java-test", map[string][]response{"gradle": {run}})
	sees(t, out, code, 1, "extra/build/reports/jacoco/test/jacocoTestReport.xml: lines 3/5 covered", "locked/build/reports/jacoco/test/jacocoTestReport.xml: lines 1/5 covered")
	javaTestNotJudged(t, out, "core/build/reports/jacoco/test/jacocoTestReport.xml")
}

// javaTestModule is a Maven module's build file with its artifact id.
func javaTestModule(id string) string {
	return "<project><modelVersion>4.0.0</modelVersion><groupId>com.example</groupId><artifactId>" + id + "</artifactId><version>1</version></project>\n"
}

// javaTestAggregate is an aggregate JaCoCo report with the given line
// totals that lists what listed holds: groups, or packages.
func javaTestAggregate(covered, missed, listed string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><report name="coverage"><sessioninfo id="s" start="1" dump="2"/>` + listed +
		`<counter type="LINE" missed="` + missed + `" covered="` + covered + `"/></report>
`
}

// javaTestListed is a report's package listing one source file.
func javaTestListed(pkg, source string) string {
	return `<package name="` + pkg + `"><sourcefile name="` + source + `"/></package>`
}

// javaTestNotJudged fails when the output names any of the reports.
func javaTestNotJudged(t *testing.T, out string, reports ...string) {
	t.Helper()
	for _, r := range reports {
		if strings.Contains(read(out), r) {
			t.Errorf("%s may not be judged:\n%s", r, out)
		}
	}
}

func Test0081_5(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, javaRepo(t, map[string]string{"pom.xml": javaPom(javaTestMaven)}), "java-test", map[string][]response{"mvn": {javaTestMavenRun(nil)}})
	sees(t, out, code, 1, "the Maven build at . has product code such as "+javaSource+" and no tests; add tests under src/test/java")
	karate := javaRepo(t, map[string]string{
		"pom.xml":                        javaPom(javaTestMaven + "<dependency><groupId>io.karatelabs</groupId><artifactId>karate-junit6</artifactId></dependency>"),
		"src/test/resources/app.feature": "Feature: app\n",
	})
	_, out, code = javaRun(t, karate, "java-test", map[string][]response{"mvn": {javaTestMavenRun(nil)}})
	sees(t, out, code, 1, "the Maven build at . has test files but ran no test", "Karate: Karate runs feature files only from a test class with a @Karate.Test method")
}

func Test0081_6(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{
		"pom.xml":                         javaPom(javaTestMaven),
		"src/test/java/example/DbIT.java": "import org.testcontainers.containers.PostgreSQLContainer;\nclass DbIT {}\n",
	})
	down := response{Match: []string{"info"}, Stderr: "Cannot connect to the Docker daemon. Is the docker daemon running?\n", Exit: 1}
	rp, out, code := javaRun(t, repo, "java-test", map[string][]response{"mvn": {javaTestMavenRun(nil)}, "docker": {down}})
	sees(t, out, code, 1, "Testcontainers needs a running Docker; start Docker or set DOCKER_HOST, then rerun")
	if calls := rp.calls("mvn"); len(calls) != 0 {
		t.Errorf("the build ran %q, want no run without Docker", calls)
	}
	// A hung Docker fails fast rather than holding the check to its limit.
	hung := response{Match: []string{"info"}, Wait: 60}
	start := time.Now()
	_, out, code = javaRun(t, repo, "java-test", map[string][]response{"mvn": {javaTestMavenRun(nil)}, "docker": {hung}})
	sees(t, out, code, 1, "Testcontainers needs a running Docker")
	if took := time.Since(start); took > 30*time.Second {
		t.Errorf("a hung Docker held the check for %s, want under 30s", took)
	}
}

func Test0081_7(t *testing.T) {
	t.Parallel()
	repo := javaRepo(t, map[string]string{"pom.xml": javaPom("")})
	_, out, code := javaRun(t, repo, "java-test", map[string][]response{"mvn": {javaTestMavenRun(nil)}})
	sees(t, out, code, 1, "JaCoCo is not in the Maven build at .; add it:")
	javaSeesSnippet(t, out, "<artifactId>jacoco-maven-plugin</artifactId>")
	noXML := response{Match: []string{"jacocoTestReport"}, Stdout: "BUILD SUCCESSFUL in 3s\n", Files: map[string]string{
		"build/test-results/test/TEST-example.AppTest.xml": javaTestPassed,
		"build/reports/jacoco/test/html/index.html":        "<html></html>\n",
	}}
	_, out, code = javaRun(t, javaRepo(t, map[string]string{"build.gradle": javaTestGradle}), "java-test", map[string][]response{"gradle": {noXML}})
	sees(t, out, code, 1, "no JaCoCo XML report after the Gradle build at . ran its tests; add:")
	javaSeesSnippet(t, out, "xml.required = true")
}

func Test0081_8(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "java-test", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "java-test")
}
