package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/pkg/actions"
)

func init() {
	register("shape-release-notes", "Shape the release PR's notes for pr-check, with plain bullets and a compare link that points at the head commit.", []string{"edited"}, shapeReleaseNotes)
	register("link-release-tracking-issue", "Link the release PR to its tracking issue, opening the issue when none exists, since every PR must close one.", []string{"edited"}, linkReleaseTrackingIssue)
	register("reopen-pr", "Close and reopen the PR in PR, so pr-check reads its edited body.", nil, reopenPR)
	register("move-major-tag", "Move the major tag of each Release Please release in OUTPUTS, v1 for the root and <component>-v1 for a component.", []string{"releases"}, moveMajorTag)
}

var (
	// compareHeading is a release heading that links a compare view.
	compareHeading = regexp.MustCompile(`^##+ \[.*/compare/`)
	// compareRange is the head end of a compare link.
	compareRange = regexp.MustCompile(`\.\.\.[^)]*\)`)
)

// shapeNotes demotes the `##` headings after "## Changelog", strips bold and
// points each compare link at head, leaving one blank line after it.
func shapeNotes(body, head string) string {
	lines := strings.Split(body, "\n")
	at := slices.Index(lines, "## Changelog")
	if at < 0 {
		return strings.TrimRight(body, "\n")
	}
	shaped, rest := slices.Clone(lines[:at+1]), lines[at+1:]
	if len(rest) > 0 && rest[0] != "" {
		shaped = append(shaped, "")
	}
	for _, line := range rest {
		shaped = append(shaped, shapeLine(line, head))
	}
	return strings.TrimRight(strings.Join(shaped, "\n"), "\n")
}

// shapeLine is one line after "## Changelog" as pr-check wants it.
func shapeLine(line, head string) string {
	if compareHeading.MatchString(line) {
		line = replaceFirst(compareRange, line, "..."+head+")")
	}
	if rest, ok := strings.CutPrefix(line, "## "); ok {
		line = "### " + rest
	}
	return strings.ReplaceAll(line, "**", "")
}

// replaceFirst replaces the first match of re in s with repl.
func replaceFirst(re *regexp.Regexp, s, repl string) string {
	loc := re.FindStringIndex(s)
	if loc == nil {
		return s
	}
	return s[:loc[0]] + repl + s[loc[1]:]
}

func shapeReleaseNotes(step actions.Step, _ []string) error {
	pr := stepPR()
	body, err := pr.field("body")
	if err != nil {
		return err
	}
	head, err := pr.field("headRefOid")
	if err != nil {
		return err
	}
	return editBody(step, pr.number, body, shapeNotes(body, head))
}

// editBody sets the PR's body to edited and writes edited=true, unless it is
// body already.
func editBody(step actions.Step, pr, body, edited string) error {
	if edited == body {
		return nil
	}
	if err := actions.Run("gh", "pr", "edit", pr, "--body", edited); err != nil {
		return err
	}
	return step.Output("edited", "true")
}

// listedIssue is one issue `gh issue list` found.
type listedIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// trackingIssue finds the issue titled title, opening it when none exists.
func trackingIssue(pr, title string) (string, error) {
	var found []listedIssue
	if err := actions.GhJSON(&found, "issue", "list", "--state", "all", "--search", `"`+title+`" in:title`, "--json", "number,title"); err != nil {
		return "", err
	}
	if at := slices.IndexFunc(found, func(i listedIssue) bool { return i.Title == title }); at >= 0 {
		return fmt.Sprint(found[at].Number), nil
	}
	url, err := actions.Gh("issue", "create", "--title", title, "--body", "Release Please owns the version and notes in #"+pr+".")
	return url[strings.LastIndex(url, "/")+1:], err
}

func linkReleaseTrackingIssue(step actions.Step, _ []string) error {
	pr := env("PR")
	issue, err := trackingIssue(pr, "Publish the release from PR #"+pr)
	if err != nil {
		return err
	}
	body, err := stepPR().field("body")
	if err != nil || strings.Contains(body, "Closes #"+issue+".") {
		return err
	}
	return editBody(step, pr, body, body+"\n\nCloses #"+issue+".")
}

func reopenPR(actions.Step, []string) error {
	pr := env("PR")
	return firstError(
		func() error { return actions.Run("gh", "pr", "close", pr) },
		func() error { return actions.Run("gh", "pr", "reopen", pr) },
	)
}

// majorRelease is one release whose major tag moves.
type majorRelease struct {
	Path     string `json:"path"`
	Tag      string `json:"tag"`
	MajorTag string `json:"major_tag"`
	SHA      string `json:"sha"`
}

// plannedReleases reads Release Please's outputs and names each release's
// major tag, failing before any tag moves when one does not fit.
func plannedReleases(outputs string) ([]majorRelease, error) {
	var o map[string]any
	if err := json.Unmarshal([]byte(outputs), &o); err != nil {
		return nil, err
	}
	paths, err := releasedPaths(o)
	plan := []majorRelease{}
	for _, path := range paths {
		r, err := planRelease(o, path)
		if err != nil {
			return nil, err
		}
		plan = append(plan, r)
	}
	return plan, err
}

// releasedPaths is the paths Release Please released, the root when unset.
func releasedPaths(o map[string]any) ([]string, error) {
	released, _ := o["paths_released"].(string)
	if released == "" {
		released = `["."]`
	}
	var paths []string
	return paths, json.Unmarshal([]byte(released), &paths)
}

// planRelease names the major tag of the release at path.
func planRelease(o map[string]any, path string) (majorRelease, error) {
	prefix := path + "--"
	if path == "." {
		prefix = ""
	}
	r := majorRelease{Path: path, Tag: outputText(o[prefix+"tag_name"]), SHA: outputText(o[prefix+"sha"])}
	major := outputText(o[prefix+"major"])
	if r.Tag == "" || major == "" || r.SHA == "" {
		return r, actions.Fail("Release Please gave no tag, major or sha for path " + path)
	}
	var err error
	r.MajorTag, err = majorTagFor(path, r.Tag, major)
	return r, err
}

// majorTagFor is v<major> for the root, and <component>-v<major> for a
// component, whose tags read <component>-vX.Y.Z.
func majorTagFor(path, tag, major string) (string, error) {
	if path == "." {
		return "v" + major, nil
	}
	cut := strings.LastIndex(tag, "-v")
	if cut < 0 {
		return "", actions.Fail("Tag " + tag + " does not fit path " + path + ": a component tags <component>-vX.Y.Z")
	}
	return tag[:cut] + "-v" + major, nil
}

// outputText is a Release Please output as text, "" when it is missing.
func outputText(v any) string {
	if v == nil {
		return ""
	}
	return text(v)
}

func moveMajorTag(step actions.Step, _ []string) error {
	plan, err := plannedReleases(env("OUTPUTS"))
	if err != nil {
		return err
	}
	for _, r := range plan {
		if err := moveTag(actions.RepoPath(), r); err != nil {
			return err
		}
	}
	var releases bytes.Buffer
	encoder := json.NewEncoder(&releases)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(plan); err != nil {
		return err
	}
	return step.Output("releases", strings.TrimSpace(releases.String()))
}

// moveTag points r's major tag at its commit, creating only a missing tag;
// any other lookup error fails by name.
func moveTag(repo string, r majorRelease) error {
	refs := repo + "/git/refs"
	found, err := actions.Call{}.Combined("gh", "api", repo+"/git/ref/tags/"+r.MajorTag, "--silent")
	switch {
	case err == nil:
		return actions.Run("gh", "api", "-X", "PATCH", refs+"/tags/"+r.MajorTag, "-f", "sha="+r.SHA, "-F", "force=true", "--silent")
	case strings.Contains(found, "Not Found"):
		return actions.Run("gh", "api", "-X", "POST", refs, "-f", "ref=refs/tags/"+r.MajorTag, "-f", "sha="+r.SHA, "--silent")
	}
	return actions.Fail("Could not read tag " + r.MajorTag + ": " + found)
}
