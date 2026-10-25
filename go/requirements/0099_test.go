package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// onboardingSkill links the skills into a new folder with the installer, as an
// agent does, and returns the text of the onboarding skill found there.
func onboardingSkill(t *testing.T) string {
	t.Helper()
	folder := filepath.Join(t.TempDir(), "skills")
	out, code := fiRun(t, t.TempDir(), nil, "--link-skills", folder)
	sees(t, out, code, 0, "fitness: skill "+filepath.Join(folder, "onboarding")+" linked to ")
	return fiFile(t, filepath.Join(folder, "onboarding", "SKILL.md"))
}

func Test0099_1(t *testing.T) {
	t.Parallel()
	skill := onboardingSkill(t)
	sees(t, skill, 0, 0, "## Install and run", "docs/ci.md#download-the-go-installer", "docs/ci.md#run-checks",
		"docs/ci.md#commit-hook", "## Read a failure", "skills/fitness/SKILL.md", "## Plans and pull requests", "fitness-check-plan-trailer/README.md")
}

// onboardingLink matches a link into the repo's main branch, with its optional heading.
var onboardingLink = regexp.MustCompile(`\]\(https://github\.com/may-journal/fitness-runner/blob/main/([^)#]+)(?:#([^)]+))?\)`)

// slugDrop holds what GitHub removes from a heading to name its anchor.
var slugDrop = regexp.MustCompile(`[^a-z0-9_ -]`)

// headingSlug is how GitHub names a heading's anchor.
func headingSlug(heading string) string {
	kept := slugDrop.ReplaceAllString(strings.ToLower(strings.TrimSpace(heading)), "")
	return strings.ReplaceAll(kept, " ", "-")
}

// hasHeading reports whether a markdown file has a heading with that anchor.
// A line inside a fenced code block is code, such as a shell comment, not a heading.
func hasHeading(text, anchor string) bool {
	fenced := false
	for _, line := range strings.Split(text, "\n") {
		fenced = fenced != isFence(line)
		if !fenced && isHeading(line, anchor) {
			return true
		}
	}
	return false
}

// isFence reports whether the line opens or closes a fenced code block.
func isFence(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func isHeading(line, anchor string) bool {
	return strings.HasPrefix(line, "#") && headingSlug(strings.TrimLeft(line, "# ")) == anchor
}

// checkSkillLink fails unless the linked file exists in the repo and holds the
// linked heading.
func checkSkillLink(t *testing.T, path, anchor string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("%s does not exist in the repo: %v", path, err)
	}
	if anchor != "" && !hasHeading(string(data), anchor) {
		t.Fatalf("%s has no heading for #%s", path, anchor)
	}
}

func Test0099_2(t *testing.T) {
	t.Parallel()
	links := onboardingLink.FindAllStringSubmatch(onboardingSkill(t), -1)
	if len(links) == 0 {
		t.Fatal("the onboarding skill links no doc")
	}
	for _, link := range links {
		checkSkillLink(t, link[1], link[2])
	}
}
