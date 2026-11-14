package repocheck

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// ADRPrefix starts the name of a check folder whose README.md is an
// architecture decision record, beside the check that keeps it.
const ADRPrefix = "adr-"

// superseded matches a status line that retires the decision, such as
// `Status: Superseded by ADR 0012` or a `## Status` section opening with
// Superseded.
var superseded = regexp.MustCompile(`(?im)(^[\s>*_-]*status[\s*_]*:[\s*_]*superseded\b)|(^#+\s*status\s*$\s*^[\s>*_-]*superseded\b)`)

// Retired reports whether an ADR's text says it is superseded, so it
// needs no check of its own.
func Retired(text string) bool {
	return superseded.MatchString(text)
}

// notCheck reports whether a checks subfolder is hidden, shared code, fixtures,
// or a superseded ADR, so it holds no check.
func notCheck(folder string) bool {
	name := filepath.Base(folder)
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || slices.Contains(skipped, name) || retiredADR(folder)
}

// retiredADR reports whether folder is an adr-NNNN folder whose README
// retires the decision and that holds no program, so it is no check.
func retiredADR(folder string) bool {
	if !strings.HasPrefix(filepath.Base(folder), ADRPrefix) || exists(filepath.Join(folder, Program)) || hasGo(folder) {
		return false
	}
	data, err := os.ReadFile(filepath.Join(folder, Readme))
	return err == nil && Retired(string(data))
}
