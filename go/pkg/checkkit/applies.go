package checkkit

import (
	"fmt"
	"regexp"
	"slices"
)

// Requirement is something a check needs beyond the documents it judges.
// A run that lacks one skips the check and says why.
type Requirement string

// The requirements a check declares in Requires.
const (
	// RequiresRepo needs the full tracked tree, not only the run's files.
	RequiresRepo Requirement = "repo"
	// RequiresDiff needs the whole change list, not only the matching files.
	RequiresDiff Requirement = "diff"
	// RequiresHistory needs the branch's commits.
	RequiresHistory Requirement = "history"
	// RequiresGitHub needs live GitHub data.
	RequiresGitHub Requirement = "github"
	// RequiresNetwork needs hosts other than GitHub, such as a package registry.
	RequiresNetwork Requirement = "network"
)

// Requirements lists every requirement this protocol defines, in order.
var Requirements = []Requirement{RequiresRepo, RequiresDiff, RequiresHistory, RequiresGitHub, RequiresNetwork}

// The Skipped reasons a run gives for a check it leaves out, which the
// release smoke test also expects.
const (
	SkipNeedsNoFile = "the repo tracks no file its needs pattern names"
	SkipNoDiff      = "requires diff, which only a run of changed files has"
)

// Known reports whether this protocol defines r.
func (r Requirement) Known() bool {
	return slices.Contains(Requirements, r)
}

// Patterns are a check's compiled Files, Exclude and Needs. A nil pattern
// was left empty.
type Patterns struct {
	files, exclude, needs *regexp.Regexp
}

// PatternsOf compiles the Files, Exclude and Needs of d's Effective fields,
// naming the field and pattern of the first that does not compile.
func PatternsOf(d Describe) (Patterns, error) {
	d = Effective(d)
	var p Patterns
	for _, field := range []struct {
		name, pattern string
		into          **regexp.Regexp
	}{{"files", d.Files, &p.files}, {"exclude", d.Exclude, &p.exclude}, {"needs", d.Needs, &p.needs}} {
		if field.pattern == "" {
			continue
		}
		re, err := regexp.Compile(field.pattern)
		if err != nil {
			return Patterns{}, fmt.Errorf("%s pattern %q does not compile: %w", field.name, field.pattern, err)
		}
		*field.into = re
	}
	return p, nil
}

// UnknownRequirement returns the first requirement of d this protocol does
// not define, or "" when it knows them all.
func UnknownRequirement(d Describe) Requirement {
	for _, r := range d.Requires {
		if !r.Known() {
			return r
		}
	}
	return ""
}

// Filters reports whether the patterns narrow which files reach the check.
func (p Patterns) Filters() bool {
	return p.files != nil || p.exclude != nil
}

// Matches reports whether path reaches the check: Files finds it, or is
// empty, and Exclude does not find it. Paths are repo-relative with `/`.
func (p Patterns) Matches(path string) bool {
	if p.files != nil && !p.files.MatchString(path) {
		return false
	}
	return p.exclude == nil || !p.exclude.MatchString(path)
}

// Reaches reports whether any of paths reaches the check.
func (p Patterns) Reaches(paths []string) bool {
	return slices.ContainsFunc(paths, p.Matches)
}

// ReachedBy reports whether a run of paths reaches the check. A nil paths is
// a run of the whole repo, which every check reaches.
func (p Patterns) ReachedBy(paths []string) bool {
	return paths == nil || !p.Filters() || p.Reaches(paths)
}

// HasNeeds reports whether the check declared a Needs pattern.
func (p Patterns) HasNeeds() bool {
	return p.needs != nil
}

// Met reports whether tracked holds a file Needs finds, or Needs is empty.
func (p Patterns) Met(tracked []string) bool {
	return p.needs == nil || slices.ContainsFunc(tracked, p.needs.MatchString)
}

// Requiring reports whether d declares r.
func Requiring(d Describe, r Requirement) bool {
	return slices.Contains(d.Requires, r)
}
