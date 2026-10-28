package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// changeErrors fails each ADR whose decision changed while no other file
// of its check did. The check's own part of its README, and the check's
// code alone, may change freely, since the decision still holds. A README
// that held no decision at the base, such as one an ADR just moved into,
// is new. A retired ADR pairs with nothing, and a run that changes
// nothing, such as a full scan, judges no pairs.
func changeErrors(root string, decisions []adr, changed []string) []string {
	alone := readmeAlone(decisions, changed)
	if len(alone) == 0 {
		return nil
	}
	base := gitx.ScopeBase(root)
	var errs []string
	for _, a := range alone {
		before := gitx.FileAt(root, base, a.readme)
		if len(missingSections(before)) == 0 && decision(before) != decision(a.text) {
			errs = append(errs, fmt.Sprintf("%s changed its decision without its check %s; change both together", a.readme, a.folder))
		}
	}
	return errs
}

// readmeAlone lists each standing ADR whose README changed while no other
// file of its check did.
func readmeAlone(decisions []adr, changed []string) []adr {
	var out []adr
	for _, a := range decisions {
		if !a.retired && slices.Contains(changed, a.readme) && !codeChanged(a, changed) {
			out = append(out, a)
		}
	}
	return out
}

// codeChanged reports whether a file of the ADR's check other than its
// README changed.
func codeChanged(a adr, changed []string) bool {
	return slices.ContainsFunc(changed, func(f string) bool {
		return strings.HasPrefix(f, a.folder+"/") && f != a.readme
	})
}

// decision is the text of an ADR's decision sections, joined.
func decision(text string) string {
	var parts []string
	for _, s := range sections {
		body, _ := section(text, s)
		parts = append(parts, body)
	}
	return strings.Join(parts, "\n")
}
