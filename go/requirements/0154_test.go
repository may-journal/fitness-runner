package requirements

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// largeTreeFiles is how many generated files the large tree adds beside a
// Java project and a components view.
const largeTreeFiles = 50000

// largeTreeChecks are the repo-state checks a large tree must keep fast.
var largeTreeChecks = []string{"build-output-untracked", "architecture-components"}

// nestedDist is a build output file deep in the large tree.
const nestedDist = "tree/0049/dist/app.js"

// rowDuration matches a results-table row's time, in milliseconds or seconds.
var rowDuration = regexp.MustCompile(`│ (\d+(?:\.\d+)?)(ms|s)\s+│\s*$`)

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
	out, code := fitness(t, repo, nil, "--all", "--policy=external", "--checks="+strings.Join(largeTreeChecks, ","))
	sees(t, out, code, 0, "All 2 checks passed")
	within(t, out, largeTreeChecks, 5*time.Second)
}

func Test0154_2(t *testing.T) {
	// Not parallel, like Test0154_1.
	repo := generatedTree(t, largeTree(), largeTreeFiles)
	write(t, repo, map[string]string{nestedDist: "export const built = true;\n"})
	git(t, repo, "add", "-f", nestedDist)
	git(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "build(tree): track a build")
	out, code := fitness(t, repo, nil, "--all", "--policy=external", "--checks="+largeTreeChecks[0])
	sees(t, out, code, 1, nestedDist)
	within(t, out, largeTreeChecks[:1], 5*time.Second)
}

// within asserts each check's row shows a time under limit.
func within(t *testing.T, out string, checks []string, limit time.Duration) {
	t.Helper()
	t.Log("\n" + out)
	for _, check := range checks {
		took := rowDuration.FindStringSubmatch(row(out, check))
		if took == nil {
			t.Fatalf("%s shows no time in its row %q", check, row(out, check))
		}
		if d, _ := time.ParseDuration(took[1] + took[2]); d >= limit {
			t.Errorf("%s took %v on %d tracked files, want under %v", check, d, largeTreeFiles, limit)
		}
	}
}
