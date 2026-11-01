package check

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo writes files, by slash-separated path, into a new git repo in a
// temporary folder, commits them, and returns its root. A unit test gives a
// check a repo that breaks its rule, or keeps it, and judges it with Judge.
func Repo(t testing.TB, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	writeFiles(t, root, files)
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "test: fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	return root
}

// writeFiles writes each file under root, creating folders as needed.
func writeFiles(t testing.TB, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// Judge runs the check on the repo at root with args and returns its verdict;
// an error fails the test.
func Judge(t testing.TB, c Check, root string, args ...string) Result {
	t.Helper()
	result, err := c.Run(&Context{Root: root, Args: args})
	if err != nil {
		t.Fatalf("the check could not judge %s: %v", root, err)
	}
	return result
}

// Test runs the check on the repo the test runs in, failing on each error line
// and logging each info note.
func Test(t testing.TB, c Check) {
	t.Helper()
	result := Judge(t, c, repoRoot(t))
	for _, note := range result.Info {
		t.Log(note)
	}
	for _, line := range result.Errors {
		t.Error(line)
	}
	if !result.Ok && len(result.Errors) == 0 {
		t.Error("the check failed without an error line")
	}
}

// repoRoot is the root of the git work tree the test runs in.
func repoRoot(t testing.TB) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("a check's test runs inside the repo it judges: git rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}
