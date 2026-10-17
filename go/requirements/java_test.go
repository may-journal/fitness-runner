// cspell:ignore mvnw

package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// javaJDK is what `java -version` prints, on stderr, for a running JDK 21.
var javaJDK = response{
	Match:  []string{"-version"},
	Stderr: "openjdk version \"21.0.2\" 2024-01-16\nOpenJDK Runtime Environment (build 21.0.2+13)\n",
}

// javaSource is a Java source file every Java example repo holds.
const javaSource = "src/main/java/App.java"

// javaPom is a Maven build file whose build declares plugins.
func javaPom(plugins string) string {
	return "<project><modelVersion>4.0.0</modelVersion><groupId>com.example</groupId>" +
		"<artifactId>app</artifactId><version>1</version><build><plugins>" + plugins + "</plugins></build></project>\n"
}

// javaParent is a Maven build that declares plugins and lists modules.
func javaParent(plugins string, modules ...string) string {
	list := "<modules>"
	for _, m := range modules {
		list += "<module>" + m + "</module>"
	}
	return strings.Replace(javaPom(plugins), "<build>", list+"</modules><build>", 1)
}

// javaRepo makes happyRepo with a Java source and files written over it.
func javaRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	all := map[string]string{javaSource: "class App {}\n"}
	for name, body := range files {
		all[name] = body
	}
	return example(t, "happyRepo", all)
}

// javaRun runs `fitness-install -- <check>` on repo with a JDK 21 and the
// build tools answering saved responses, and JAVA_HOME unset so the JDK on
// PATH is the one that runs.
func javaRun(t *testing.T, repo, check string, saved map[string][]response) (replays, string, int) {
	t.Helper()
	all := map[string][]response{"java": {javaJDK}}
	for name, responses := range saved {
		all[name] = responses
	}
	rp := standIn(t, all)
	out, code := fitness(t, repo, append(rp.env, "JAVA_HOME="), check)
	return rp, out, code
}

// javaWrapper puts the stand-in at the root of repo as the build wrapper
// name, so a run of the wrapper answers from its saved responses.
func javaWrapper(t *testing.T, repo, name string) {
	t.Helper()
	self, err := os.Executable()
	mustDo(t, err)
	mustDo(t, os.Symlink(self, filepath.Join(repo, name)))
}

// javaSeesSnippet asserts the user saw a build file snippet, which the
// results table may break inside a long word, by comparing without spaces.
func javaSeesSnippet(t *testing.T, out, snippet string) {
	t.Helper()
	squeeze := func(s string) string { return strings.Join(strings.Fields(read(s)), "") }
	if !strings.Contains(squeeze(out), squeeze(snippet)) {
		t.Errorf("output is missing the snippet %q:\n%s", snippet, out)
	}
}

// javaCalledOnce asserts the named build tool ran exactly once.
func javaCalledOnce(t *testing.T, rp replays, name string) {
	t.Helper()
	if calls := rp.calls(name); len(calls) != 1 {
		t.Errorf("%s calls = %q, want one", name, calls)
	}
}
