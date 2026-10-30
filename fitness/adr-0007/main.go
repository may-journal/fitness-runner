// Command adr-0007 keeps ADR 0007: CHANGELOG.md stays frozen at 1.14.0, and
// Release Please writes no changelog.
package main

import (
	"regexp"

	"github.com/may-journal/fitness-runner/go/check"
)

// newest matches the first release heading of CHANGELOG.md.
var newest = regexp.MustCompile(`(?m)^## \[([^\]]+)\]`)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	text, err := ctx.Read("CHANGELOG.md")
	if err != nil {
		return check.Result{}, err
	}
	var found []check.Finding
	if m := newest.FindStringSubmatch(text); m == nil || m[1] != "1.14.0" {
		found = append(found, check.Finding{File: "CHANGELOG.md", Rule: "frozen-changelog", Message: "its newest release is not 1.14.0, where it froze (ADR 0007, decision 6)",
			Fix: "add a changelog/ entry file and leave CHANGELOG.md as it is"})
	}
	more, err := skipsChangelog(ctx)
	return check.Found(2, append(found, more...)...), err
}

// skipsChangelog fails a Release Please config that lets a release write
// CHANGELOG.md.
func skipsChangelog(ctx *check.Context) ([]check.Finding, error) {
	var release struct {
		Packages map[string]struct {
			SkipChangelog bool `json:"skip-changelog"`
		} `json:"packages"`
	}
	if _, err := ctx.ReadJSON("release-please-config.json", &release); err != nil || release.Packages["."].SkipChangelog {
		return nil, err
	}
	return []check.Finding{{File: "release-please-config.json", Rule: "skip-changelog", Message: "lets a release write CHANGELOG.md (ADR 0007, decision 4)",
		Fix: `set "skip-changelog": true`}}, nil
}
