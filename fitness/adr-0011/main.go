// Command adr-0011 keeps ADR 0011: each skill is a kebab-case folder with
// a SKILL.md named for it, and every bundle carries the skills.
package main

import (
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// kebab matches a kebab-case name.
var kebab = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	folders := skills(ctx)
	var found []check.Finding
	for _, name := range folders {
		more, err := skill(ctx, name)
		if err != nil {
			return check.Result{}, err
		}
		found = append(found, more...)
	}
	bundles, err := ctx.Read(".goreleaser-bundles.yml")
	if err != nil {
		return check.Result{}, err
	}
	if !strings.Contains(bundles, "skills/**/*") {
		found = append(found, check.Finding{File: ".goreleaser-bundles.yml", Rule: "bundle-skills", Message: "no longer carries skills/ (ADR 0011, decision 3)",
			Fix: "keep skills/**/* in the bundle's files"})
	}
	return check.Found(len(folders)+1, found...), nil
}

// skills lists the folder names under skills/, once each.
func skills(ctx *check.Context) []string {
	var out []string
	for _, f := range ctx.Files() {
		if name := skillFolder(f); name != "" && !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

// skillFolder is the skill folder a file under skills/ sits in, or "".
func skillFolder(f string) string {
	rest, ok := strings.CutPrefix(f, "skills/")
	name, _, nested := strings.Cut(rest, "/")
	if !ok || !nested {
		return ""
	}
	return name
}

// skill judges one skill folder's name and its SKILL.md.
func skill(ctx *check.Context, name string) ([]check.Finding, error) {
	var found []check.Finding
	if !kebab.MatchString(name) {
		found = append(found, check.Finding{File: "skills/" + name, Rule: "kebab-skill", Message: "is not kebab case (ADR 0011, decision 1)", Fix: "rename the folder"})
	}
	text, err := ctx.Read("skills/" + name + "/SKILL.md")
	if err != nil && !ctx.Exists("skills/"+name+"/SKILL.md") {
		err = nil
	}
	if !strings.Contains(text, "\nname: "+name+"\n") {
		found = append(found, check.Finding{File: "skills/" + name + "/SKILL.md", Rule: "skill-name", Message: "is missing or names another skill (ADR 0011, decision 1)",
			Fix: "give the folder a SKILL.md whose name is " + name})
	}
	return found, err
}
