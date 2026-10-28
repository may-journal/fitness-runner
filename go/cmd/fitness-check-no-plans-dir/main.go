package check

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

func Main() {
	checkkit.Main(checkkit.Check{
		Describe: checkkit.Describe{Name: "no-plans-dir", Scope: checkkit.ScopeFiles},
		Run:      run,
	})
}

// Where each kind of temporary work belongs instead of the repo.
const (
	toPlanIssue  = "open a `Plan`-labeled GitHub Issue and put the plan there"
	toPlanDesign = "post it as a comment on the Plan Issue that it designs"
	toPRComment  = "post it as a comment on the pull request it reviews"
)

// artifactFolders maps each folder name, lowercased, to where its files go.
var artifactFolders = map[string]string{
	"plans":            toPlanIssue,
	"mockups":          toPlanDesign,
	"validation":       toPRComment,
	"review-artifacts": toPRComment,
}

// prArtifact matches a file named for one pull request, such as
// `pr-12-before.png`.
var prArtifact = regexp.MustCompile(`(?i)^pr-\d+-`)

// mockupFile matches a file named as a mockup, such as `home-mockup.png`.
var mockupFile = regexp.MustCompile(`(?i)-mockup\.[^/]+$`)

// planChecklist matches a Plan's `## What needs to happen` heading followed,
// past any blank lines, by a checklist item.
var planChecklist = regexp.MustCompile(`(?mi)^## What needs to happen[ \t]*\n(?:[ \t]*\n)*[ \t]*[-*] \[[ xX]\] `)

// finding is one offending path, the rule it matched, and where it belongs.
type finding struct {
	path, rule, fix string
}

func run(root string, _ []string) (checkkit.Result, error) {
	files, err := walkfs.AllFiles(root)
	if err != nil {
		return checkkit.Fail(0, "cannot list the repo's files: "+err.Error()), nil
	}
	var errs []string
	for _, f := range files {
		if found, ok := judge(root, f); ok {
			errs = append(errs, found.path+": "+found.rule+"; "+found.fix+", not a file in the repo")
		}
	}
	if len(errs) == 0 {
		return checkkit.Pass(len(files)), nil
	}
	return checkkit.Fail(len(files), errs...), nil
}

// judge reports the first rule rel breaks: an artifact folder, an artifact
// file name, or a Plan checklist in Markdown outside `.github/`.
func judge(root, rel string) (finding, bool) {
	if found, ok := inArtifactFolder(rel); ok {
		return found, true
	}
	if found, ok := artifactName(rel); ok {
		return found, true
	}
	if holdsPlan(root, rel) {
		return finding{rel, "a `## What needs to happen` checklist is a plan", toPlanIssue}, true
	}
	return finding{}, false
}

// inArtifactFolder reports the first folder of rel that holds temporary work.
func inArtifactFolder(rel string) (finding, bool) {
	dirs := strings.Split(rel, "/")
	for _, dir := range dirs[:len(dirs)-1] {
		if fix, ok := artifactFolders[strings.ToLower(dir)]; ok {
			return finding{rel, "a `" + strings.ToLower(dir) + "` folder holds temporary work", fix}, true
		}
	}
	return finding{}, false
}

// artifactName reports whether rel's file name marks a plan, a mockup, or
// one pull request's artifact.
func artifactName(rel string) (finding, bool) {
	name := path.Base(rel)
	switch {
	case strings.HasSuffix(strings.ToLower(name), "-plan.md"):
		return finding{rel, "a `*-plan.md` file is a plan", toPlanIssue}, true
	case mockupFile.MatchString(name):
		return finding{rel, "a `*-mockup.*` file is a design mockup", toPlanDesign}, true
	case prArtifact.MatchString(name):
		return finding{rel, "a `pr-<number>-*` file belongs to one pull request", toPRComment}, true
	}
	return finding{}, false
}

// holdsPlan reports whether rel is a Markdown file outside `.github/`, where
// issue templates live, holding a Plan's checklist. A file that cannot be
// read holds none; the checks that read Markdown name it.
func holdsPlan(root, rel string) bool {
	if !strings.HasSuffix(strings.ToLower(rel), ".md") || strings.HasPrefix(rel, ".github/") {
		return false
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil && planChecklist.Match(content)
}
