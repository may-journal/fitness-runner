package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/repocheck"
)

// changeErrors fails each changed ADR whose check stayed unchanged, and each
// changed check file whose ADR stayed unchanged. A retired ADR pairs with
// nothing, and a run that changes nothing, such as a full scan, judges no
// pairs.
func changeErrors(decisions []adr, changed []string) []string {
	var errs []string
	for _, a := range decisions {
		if !a.retired {
			errs = append(errs, adrChange(a, changed)...)
		}
	}
	for _, f := range changed {
		errs = append(errs, checkChange(decisions, f, changed)...)
	}
	return errs
}

// adrChange fails a changed ADR when no file of its check changed.
func adrChange(a adr, changed []string) []string {
	folder := repocheck.Folder + "/adr-" + a.number + "/"
	if !slices.Contains(changed, a.path) || slices.ContainsFunc(changed, func(f string) bool { return strings.HasPrefix(f, folder) }) {
		return nil
	}
	return []string{fmt.Sprintf("%s changed without its check %s; change both together", a.path, strings.TrimSuffix(folder, "/"))}
}

// checkChange fails a changed file of an ADR check when its ADR did not
// change.
func checkChange(decisions []adr, file string, changed []string) []string {
	i := slices.IndexFunc(decisions, func(a adr) bool { return a.number == checkNumber(file) })
	if i < 0 || decisions[i].retired || slices.Contains(changed, decisions[i].path) {
		return nil
	}
	return []string{fmt.Sprintf("%s changed without its ADR %s; change both together", file, decisions[i].path)}
}

// checkNumber is the ADR number of a file inside a fitness/adr-NNNN
// folder, or "".
func checkNumber(file string) string {
	rest, inFolder := strings.CutPrefix(file, repocheck.Folder+"/")
	folder, _, nested := strings.Cut(rest, "/")
	n, isADR := adrNumber(folder)
	if !inFolder || !nested || !isADR {
		return ""
	}
	return n
}
