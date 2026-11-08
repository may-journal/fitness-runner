package requirements

import (
	"strings"
	"testing"
)

// largeTreeFiles is how many generated files the large tree adds beside a
// Java project and a components view.
const largeTreeFiles = 50000

// largeTreeChecks are the repo-state checks a large tree must keep fast.
var largeTreeChecks = []string{"build-output-untracked", "architecture-components"}

// nestedDist is a build output file deep in the large tree.
const nestedDist = "tree/0049/dist/app.js"

// largeTree is aoFixture with a Java project, a TypeScript source the store
// owns and an ignored dist folder.
func largeTree() map[string]string {
	return with(aoFixture(), map[string]string{
		"pom.xml":            javaPom(""),
		".gitignore":         "dist/\n",
		"src/store/index.ts": "import { save } from './save';\n\nexport const ready = save;\n",
		"src/store/save.ts":  "export const save = true;\n",
	})
}

func Test0154_1(t *testing.T) {
	// Not parallel: a 50,000-file tree built beside other tests would time their disk load.
	repo := generatedTree(t, largeTree(), largeTreeFiles)
	write(t, repo, map[string]string{"dist/index.js": "export const built = true;\n"})
	path, calls := loggedGit(t)
	out, code := fitness(t, repo, []string{path}, "--all", "--policy=external", "--checks="+strings.Join(largeTreeChecks, ","))
	sees(t, out, code, 0, "All 2 checks passed")
	fewGitCalls(t, calls)
}

func Test0154_2(t *testing.T) {
	// Not parallel, like Test0154_1.
	repo := generatedTree(t, largeTree(), largeTreeFiles)
	write(t, repo, map[string]string{nestedDist: "export const built = true;\n"})
	git(t, repo, "add", "-f", nestedDist)
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "build(tree): track a build")
	path, calls := loggedGit(t)
	out, code := fitness(t, repo, []string{path}, "--all", "--policy=external", "--checks="+largeTreeChecks[0])
	sees(t, out, code, 1, nestedDist)
	fewGitCalls(t, calls)
}

// maxLargeTreeGitCalls bounds the git calls of a whole run on the large tree,
// runner included, far below one per tracked file.
const maxLargeTreeGitCalls = 20

// fewGitCalls asserts the run started git a few times in all, with one
// `git ls-files` for build-output-untracked, not one per file.
func fewGitCalls(t *testing.T, calls string) {
	t.Helper()
	n := occurrences(calls, "\n")
	t.Logf("git started %d times", n)
	if n == 0 || n > maxLargeTreeGitCalls {
		t.Errorf("the run started git %d times on %d tracked files, want 1 to %d, none per file", n, largeTreeFiles, maxLargeTreeGitCalls)
	}
}
