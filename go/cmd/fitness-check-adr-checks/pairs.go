package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/adr"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// changeErrors fails each README that now links another ADR issue while no
// other file of its check changed. The package doc's Rules list the
// exceptions.
func changeErrors(root string, decisions []record, changed []string) []string {
	alone := readmeAlone(decisions, changed)
	if len(alone) == 0 {
		return nil
	}
	base := gitx.ScopeBase(root)
	var errs []string
	for _, r := range alone {
		before := adr.LinkedIssue(gitx.FileAt(root, base, r.readme))
		if before != "" && before != r.issue {
			errs = append(errs, fmt.Sprintf("%s changed its ADR issue without its check %s; change both together", r.readme, r.folder))
		}
	}
	return errs
}

// readmeAlone lists each standing ADR whose README changed while no other
// file of its check did.
func readmeAlone(decisions []record, changed []string) []record {
	var out []record
	for _, r := range decisions {
		if !r.retired && slices.Contains(changed, r.readme) && !codeChanged(r, changed) {
			out = append(out, r)
		}
	}
	return out
}

// codeChanged reports whether a file of the ADR's check other than its
// README changed.
func codeChanged(r record, changed []string) bool {
	return slices.ContainsFunc(changed, func(f string) bool {
		return strings.HasPrefix(f, r.folder+"/") && f != r.readme
	})
}
