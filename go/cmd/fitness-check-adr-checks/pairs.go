package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/repocheck"
)

// changeErrors fails each changed ADR whose check stayed unchanged. A
// check may change alone, as when it is rewritten, since its decision
// holds. A retired ADR pairs with nothing, and a run that changes nothing,
// such as a full scan, judges no pairs.
func changeErrors(decisions []adr, changed []string) []string {
	var errs []string
	for _, a := range decisions {
		if !a.retired {
			errs = append(errs, adrChange(a, changed)...)
		}
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
