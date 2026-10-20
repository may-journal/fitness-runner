// Command fitness-check-changelog judges the changelog/*.md entry files,
// the only changelog source. Each must hold one dated
// "### yyyy.mm.dd.HHMM" heading and its bullets, and its name must start
// with that heading's date. CHANGELOG.md, in any shape or absent, is not
// judged, and neither is the package version.
package check

import (
	"github.com/may-journal/fitness-runner/go/internal/changelog"
	"github.com/may-journal/fitness-runner/go/internal/checkkit"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "changelog"},
		Run:      run,
	})
}

// run judges every entry file; a repo without any passes with none checked.
func run(root string, _ []string) (checkkit.Result, error) {
	entries, err := changelog.ReadEntries(root)
	if err != nil {
		return checkkit.Result{}, err
	}
	if errs := entryErrors(entries); len(errs) > 0 {
		return checkkit.Fail(len(entries), errs...), nil
	}
	return checkkit.Pass(len(entries)), nil
}

// entryErrors names each entry file that holds other than one dated heading,
// or whose name does not start with that heading's date.
func entryErrors(entries []changelog.Entry) []string {
	var errs []string
	for _, e := range entries {
		if !e.Shaped() {
			errs = append(errs, e.Path+" must hold one ### yyyy.mm.dd.HHMM heading and its bullets, nothing else")
		} else if msg := changelog.NameError(e.Path, e.Text); msg != "" {
			errs = append(errs, msg)
		}
	}
	return errs
}
