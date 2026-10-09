package requirements

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stampEntry is a changelog whose newest entry sits under an old heading.
const stampEntry = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n" +
	"### 2000.01.01.0000\n\n- Docs: add the usage note.\n- Docs: show how to run it.\n- Docs: link the usage note.\n\n" +
	"### 2026.10.08.1400\n\n- Docs: add the readme.\n- Docs: add the changelog.\n- Docs: add the license note.\n"

// stampReadme adds the usage note the stamped entry describes.
const stampReadme = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# App\n\nShort and clean.\n\n" +
	"## Usage\n\nRun the usage note to show how to link it.\n"

// stampCommit writes files in a hooked happyRepo, stages and commits them as
// a user does, and returns the repo and every heading the commit time may show.
func stampCommit(t *testing.T, files map[string]string) (string, []string) {
	t.Helper()
	repo, env := hooked(t, nil)
	write(t, repo, files)
	user(t, repo, env, "add", "README.md", "CHANGELOG.md")
	before := time.Now()
	out, code := user(t, repo, env, "commit", "-m", "docs(readme): add the usage note")
	sees(t, out, code, 0)
	return repo, []string{stampHeading(before), stampHeading(time.Now())}
}

// stampHeading is the heading the stamp writes for a moment.
func stampHeading(at time.Time) string {
	return "### " + at.Format("2006.01.02.1504")
}

// stampCommitted returns CHANGELOG.md as the last commit holds it.
func stampCommitted(t *testing.T, repo string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "show", "HEAD:CHANGELOG.md").Output()
	mustDo(t, err)
	return string(out)
}

// stampShows asserts text holds one of the headings the commit time may show.
func stampShows(t *testing.T, label, text string, headings []string) {
	t.Helper()
	if !strings.Contains(text, headings[0]) && !strings.Contains(text, headings[1]) {
		t.Errorf("%s does not show the commit time %v:\n%s", label, headings, text)
	}
}

func Test0063_1(t *testing.T) {
	t.Parallel()
	repo, headings := stampCommit(t, map[string]string{"CHANGELOG.md": stampEntry, "README.md": stampReadme})
	stampShows(t, "the committed CHANGELOG", stampCommitted(t, repo), headings)
	mine, _ := os.ReadFile(filepath.Join(repo, "CHANGELOG.md"))
	stampShows(t, "my CHANGELOG", string(mine), headings)
}

func Test0063_2(t *testing.T) {
	t.Parallel()
	repo, headings := stampCommit(t, map[string]string{"CHANGELOG.md": stampEntry, "README.md": stampReadme})
	got := stampCommitted(t, repo)
	stampShows(t, "the committed CHANGELOG", got, headings)
	if strings.Contains(got, "2000.01.01.0000") || !strings.Contains(got, "### 2026.10.08.1400") {
		t.Errorf("only the newest heading may change:\n%s", got)
	}
}

func Test0063_3(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	write(t, repo, map[string]string{"CHANGELOG.md": stampEntry})
	out, code := user(t, repo, env, "commit", "--allow-empty", "-m", "docs(readme): touch nothing")
	sees(t, out, code, 0)
	if got := stampCommitted(t, repo); strings.Contains(got, "Docs: add the usage note.") {
		t.Errorf("the unstaged CHANGELOG edit was committed:\n%s", got)
	}
	if mine, _ := os.ReadFile(filepath.Join(repo, "CHANGELOG.md")); string(mine) != stampEntry {
		t.Errorf("my unstaged CHANGELOG was rewritten:\n%s", mine)
	}
}
