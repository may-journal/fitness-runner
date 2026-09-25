// Command fitness-check-no-plans-dir guards the plans-to-Issues migration: it
// fails when any file exists under docs/plans/, because plans now live as
// `Plan`-labeled GitHub Issues, not checked-in files. It points the author to
// open a Plan Issue instead. Wired into the file suite, so the pre-commit hook
// catches a docs/plans file before it lands.
package main

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "no-plans-dir"},
		Run:      run,
	})
}

const guidance = ": plans live as `Plan`-labeled GitHub Issues, not files — open a Plan Issue instead of adding to docs/plans/"

func run(root string, _ []string) (checkkit.Result, error) {
	offenders := plansFiles(root)
	if len(offenders) == 0 {
		return checkkit.Pass(0), nil
	}
	errs := make([]string, len(offenders))
	for i, f := range offenders {
		errs[i] = f + guidance
	}
	return checkkit.Fail(len(offenders), errs...), nil
}

// plansFiles returns the slash-separated repo-relative paths of every file
// under docs/plans/, sorted. A missing directory yields none.
func plansFiles(root string) []string {
	dir := filepath.Join(root, "docs", "plans")
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if rel, e := filepath.Rel(root, path); e == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}
