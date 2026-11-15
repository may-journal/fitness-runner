// cspell:ignore mvnw

package requirements

import (
	"testing"
)

// javaJDK is what `java -version` prints, on stderr, for a running JDK 21.
var javaJDK = response{
	Match:  []string{"-version"},
	Stderr: "openjdk version \"21.0.2\" 2024-01-16\nOpenJDK Runtime Environment (build 21.0.2+13)\n",
}

// javaSource is a Java source file every Java example repo holds.
const javaSource = "src/main/java/App.java"

// javaRepo makes happyRepo with a Java source and files written over it.
func javaRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	all := map[string]string{javaSource: "class App {}\n"}
	for name, body := range files {
		all[name] = body
	}
	return example(t, "happyRepo", all)
}

// javaRun runs `fitness-install -- <check>` on repo with a JDK 21 and build
// tools answering saved responses, JAVA_HOME unset so the JDK on PATH runs.
// env adds to the run's environment.
func javaRun(t *testing.T, repo, check string, saved map[string][]response, env ...string) (replays, string, int) {
	t.Helper()
	all := map[string][]response{"java": {javaJDK}}
	for name, responses := range saved {
		all[name] = responses
	}
	rp := standIn(t, all)
	out, code := fitness(t, repo, append(append(rp.env, "JAVA_HOME="), env...), check)
	return rp, out, code
}
