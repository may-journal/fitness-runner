package acceptance

import (
	"strings"
	"testing"
)

// Passing is the verdict a check prints when it passes on one file.
const Passing = `{"ok": true, "errors": [], "filesChecked": 1}`

// HouseReadme is the README a repo's own check needs to run.
const HouseReadme = "# House rule\n\nKeeps the house in order. Each room stays tidy.\n\n## Fix\n\nTidy the room the error names.\n"

// SharedScan runs every check on a repo holding files, and returns what
// fitness shows. The repo's own scan-reader check prints the test scan
// the runner shares.
func SharedScan(t *testing.T, files map[string]string) string {
	t.Helper()
	files["fitness/scan-reader/README.md"] = HouseReadme
	files["fitness/scan-reader/check"] = "#!/bin/sh\n[ \"$1\" = --describe ] && { echo '{\"name\": \"scan-reader\", \"tests\": true}'; exit 0; }\n" +
		"cat \"$FITNESS_TEST_SCAN\" >&2\necho '" + Passing + "'\n"
	out, _ := Fitness(t, Example(t, "happyRepo", files), nil, "--all")
	return out
}

// Wants fails for each fragment the shared scan out lacks.
func Wants(t *testing.T, out string, fragments ...string) {
	t.Helper()
	for _, want := range fragments {
		if !strings.Contains(out, want) {
			t.Errorf("the shared scan must hold %s, got:\n%s", want, out)
		}
	}
}
