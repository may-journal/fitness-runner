package requirements

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/checkkit"
	"github.com/may-journal/fitness-runner/go/pkg/lang"
)

// catalogRows names every check a run of the suite's installed checks
// reports, separated by spaces, which many requirements count. It follows
// the install, so a check fitness-checks ships later counts with no edit here.
var catalogRows = strings.Join(installedCatalog(os.Getenv(checkkit.ChecksDirEnv)), " ")

// installedCatalog is the engine's catalog for the checks in dir: each
// installed check the engine does not ignore, plus local-branch, sorted.
func installedCatalog(dir string) []string {
	names := []string{"local-branch"}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		name, ok := strings.CutPrefix(e.Name(), checkkit.BinPrefix)
		if ok && !slices.Contains(names, name) && !slices.Contains(lang.Ignored(), name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// allPassed is the summary of a run where every installed check passed.
func allPassed() string {
	return fmt.Sprintf("All %d checks passed", len(strings.Fields(catalogRows)))
}

// passedOf is the summary count of a run where all but notPassed installed
// checks passed.
func passedOf(notPassed int) string {
	n := len(strings.Fields(catalogRows))
	return fmt.Sprintf("%d of %d checks passed", n-notPassed, n)
}
