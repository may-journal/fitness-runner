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
	t.Parallel()
	repo := generatedTree(t, largeTree(), largeTreeFiles)
	write(t, repo, map[string]string{"dist/index.js": "export const built = true;\n"})
	out, code := fitness(t, repo, nil, "--all", "--policy=external", "--checks="+strings.Join(largeTreeChecks, ","))
	passesFast(t, out, code, largeTreeChecks)
}
