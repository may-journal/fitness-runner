package requirements

import (
	"path/filepath"
	"strings"
	"testing"
)

// checkstyleMaven declares Checkstyle in a Maven build.
const checkstyleMaven = "<plugin><groupId>org.apache.maven.plugins</groupId><artifactId>maven-checkstyle-plugin</artifactId></plugin>"

// checkstyleConfig is where the Checkstyle config sits.
const checkstyleConfig = "config/checkstyle/checkstyle.xml"

// checkstyleCeiling is a Checkstyle config that caps complexity.
const checkstyleCeiling = `<?xml version="1.0"?>
<module name="Checker">
  <module name="TreeWalker">
    <module name="CyclomaticComplexity"><property name="max" value="5"/></module>
  </module>
</module>
`

// checkstyleNoCeiling is a Checkstyle config with rules but no ceiling.
const checkstyleNoCeiling = `<?xml version="1.0"?>
<module name="Checker">
  <module name="TreeWalker">
    <module name="MethodName"/>
  </module>
</module>
`

// checkstyleClean is Maven's answer for a Checkstyle run with no finding.
var checkstyleClean = response{
	Match:  []string{"checkstyle:check"},
	Stdout: "[INFO] You have 0 Checkstyle violations.\n[INFO] BUILD SUCCESS\n",
}

// checkstyleWarning is Maven's answer for one warning in the Java source of
// repo, printed with its absolute path as the plugin does.
func checkstyleWarning(repo string) response {
	return response{
		Match: []string{"checkstyle:check", "checkstyle.violationSeverity=warning"},
		Stdout: "[INFO] --- checkstyle:3.6.0:check (default-cli) @ app ---\n" +
			"[WARN] " + filepath.Join(repo, javaSource) + ":[3,5] (naming) MethodName: Name 'Foo' must match pattern '^[a-z][a-zA-Z0-9]*$'.\n" +
			"[ERROR] Failed to execute goal org.apache.maven.plugins:maven-checkstyle-plugin:3.6.0:check (default-cli) on project app: You have 1 Checkstyle violation. -> [Help 1]\n",
		Exit: 1,
	}
}

// checkstyleRepo is a Maven repo declaring plugins, with files written over it.
func checkstyleRepo(t *testing.T, plugins string, files map[string]string) string {
	t.Helper()
	all := map[string]string{"pom.xml": javaPom(plugins)}
	for name, body := range files {
		all[name] = body
	}
	return javaRepo(t, all)
}

func Test0079_1(t *testing.T) {
	t.Parallel()
	repo := checkstyleRepo(t, checkstyleMaven, map[string]string{checkstyleConfig: checkstyleCeiling})
	_, out, code := javaRun(t, repo, "checkstyle", map[string][]response{"mvn": {checkstyleWarning(repo)}})
	sees(t, out, code, 1, javaSource+":3:5 - Name 'Foo' must match pattern '^[a-z][a-zA-Z0-9]*$' (MethodName)")
	// Gradle prints a warning only when its task runs, never when it holds
	// the task up to date, so each task must rerun.
	repo = javaRepo(t, map[string]string{"build.gradle": "plugins { id 'java'; id 'checkstyle' }\n", checkstyleConfig: checkstyleCeiling})
	rp, out, code := javaRun(t, repo, "checkstyle", map[string][]response{"gradle": {checkstyleGradleWarning(repo)}})
	sees(t, out, code, 1, javaSource+":3:5 - Name 'Foo' must match pattern '^[a-z][a-zA-Z0-9]*$' (MethodName)")
	if !rp.called("gradle", "checkstyleMain --rerun", "checkstyleTest --rerun") {
		t.Errorf("gradle calls = %q, want both Checkstyle tasks forced to rerun", rp.calls("gradle"))
	}
}

// checkstyleGradleWarning is Gradle's answer for one Checkstyle warning in
// repo's Java source, passing the build because warnings do not fail it.
func checkstyleGradleWarning(repo string) response {
	return response{
		Match: []string{"checkstyleMain"},
		Stdout: "> Task :checkstyleMain\n" +
			"[ant:checkstyle] [WARN] " + filepath.Join(repo, javaSource) + ":3:5: Name 'Foo' must match pattern '^[a-z][a-zA-Z0-9]*$'. [MethodName]\n" +
			"\nBUILD SUCCESSFUL in 3s\n",
	}
}

func Test0079_2(t *testing.T) {
	t.Parallel()
	repo := checkstyleRepo(t, checkstyleMaven, map[string]string{
		checkstyleConfig:                    checkstyleCeiling,
		"config/checkstyle/strict.xml":      checkstyleCeiling,
		"src/test/resources/no-ceiling.xml": checkstyleNoCeiling,
	})
	clean := response{Match: []string{"checkstyle:check", "checkstyle.config.location=", "/" + checkstyleConfig}, Stdout: checkstyleClean.Stdout}
	_, out, code := javaRun(t, repo, "checkstyle", map[string][]response{"mvn": {clean}})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "checkstyle", "1")
}

func Test0079_3(t *testing.T) {
	t.Parallel()
	repo := checkstyleRepo(t, checkstyleMaven, map[string]string{checkstyleConfig: checkstyleNoCeiling})
	_, out, code := javaRun(t, repo, "checkstyle", map[string][]response{"mvn": {checkstyleClean}})
	sees(t, out, code, 1, checkstyleConfig+" sets no complexity ceiling; add inside TreeWalker:")
	javaSeesSnippet(t, out, `<module name="CyclomaticComplexity">`)
}

func Test0079_4(t *testing.T) {
	t.Parallel()
	repo := checkstyleRepo(t, checkstyleMaven, nil)
	_, out, code := javaRun(t, repo, "checkstyle", map[string][]response{"mvn": {checkstyleClean}})
	sees(t, out, code, 1, "no Checkstyle config sets a complexity ceiling; add "+checkstyleConfig+" holding")
}

func Test0079_5(t *testing.T) {
	t.Parallel()
	repo := checkstyleRepo(t, "", map[string]string{checkstyleConfig: checkstyleCeiling})
	_, out, code := javaRun(t, repo, "checkstyle", map[string][]response{"mvn": {checkstyleClean}})
	sees(t, out, code, 1, "Checkstyle is not in the Maven build at .; add it:")
	javaSeesSnippet(t, out, "<artifactId>maven-checkstyle-plugin</artifactId>")
}

// checkstyleHinted reads, from fitness's output, the config the hint asks
// a repo without one to add.
func checkstyleHinted(t *testing.T, out string) string {
	t.Helper()
	_, after, ok := strings.Cut(read(out), checkstyleConfig+" holding ")
	config, _, found := strings.Cut(after, "</module></module></module>")
	if !ok || !found {
		t.Fatalf("output gives no config to add:\n%s", out)
	}
	return config + "</module></module></module>\n"
}

func Test0079_7(t *testing.T) {
	t.Parallel()
	_, out, _ := javaRun(t, checkstyleRepo(t, checkstyleMaven, nil), "checkstyle", map[string][]response{"mvn": {checkstyleClean}})
	hinted := checkstyleHinted(t, out)
	if !strings.HasPrefix(hinted, "<!DOCTYPE module PUBLIC \"-//Checkstyle//DTD Checkstyle Configuration 1.3//EN\"") {
		t.Errorf("the hinted config lacks the document type line Maven needs:\n%s", hinted)
	}
	repo := checkstyleRepo(t, checkstyleMaven, map[string]string{checkstyleConfig: hinted})
	_, out, code := javaRun(t, repo, "checkstyle", map[string][]response{"mvn": {checkstyleClean}})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0079_6(t *testing.T) {
	t.Parallel()
	_, out, code := javaRun(t, example(t, "happyRepo", nil), "checkstyle", nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "checkstyle")
}
