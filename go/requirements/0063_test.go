package requirements

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// stampReadme adds the usage note the stamped entry describes.
const stampReadme = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# App\n\nShort and clean.\n\n" +
	"## Usage\n\nRun the usage note to show how to link it.\n"

// stampHeading is the heading the stamp writes for a moment.
func stampHeading(at time.Time) string {
	return "### " + at.Format("2006.01.02.1504")
}

// stampShows asserts text holds one of the headings the commit time may show.
func stampShows(t *testing.T, label, text string, headings []string) {
	t.Helper()
	if !strings.Contains(text, headings[0]) && !strings.Contains(text, headings[1]) {
		t.Errorf("%s does not show the commit time %v:\n%s", label, headings, text)
	}
}

func Test0063_5(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	write(t, repo, map[string]string{"changelog/usage-note.md": entryFileText("2000.01.01.0000", "Docs: add the usage note."), "README.md": stampReadme})
	user(t, repo, env, "add", "README.md", "changelog/usage-note.md")
	before := time.Now()
	out, code := user(t, repo, env, "commit", "-m", "docs(readme): add the usage note")
	sees(t, out, code, 0)
	got, err := exec.Command("git", "-C", repo, "show", "HEAD:changelog/usage-note.md").Output()
	mustDo(t, err)
	stampShows(t, "the committed entry file", string(got), []string{stampHeading(before), stampHeading(time.Now())})
}

// stampFirst is the time a branch's earlier commit stamped its entry.
const stampFirst = "2026.10.09.0900"

func Test0063_6(t *testing.T) {
	t.Parallel()
	repo, env := hooked(t, nil)
	git(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	git(t, repo, "checkout", "-q", "-b", "work")
	first := entryFileText(stampFirst, "Docs: add the usage note.")
	write(t, repo, map[string]string{"changelog/usage-note.md": first, "README.md": stampReadme})
	user(t, repo, env, "add", "README.md", "changelog/usage-note.md")
	out, code := user(t, repo, env, "commit", "--no-verify", "-m", "docs(readme): add the usage note")
	sees(t, out, code, 0)
	write(t, repo, map[string]string{"changelog/usage-note.md": first + "- Docs: show the usage example to run.\n", "README.md": stampReadme + "\nRun the usage example to show it.\n"})
	user(t, repo, env, "add", "README.md", "changelog/usage-note.md")
	out, code = user(t, repo, env, "commit", "-m", "docs(readme): add the usage example")
	sees(t, out, code, 0)
	got, err := exec.Command("git", "-C", repo, "show", "HEAD:changelog/usage-note.md").Output()
	mustDo(t, err)
	if !strings.Contains(string(got), "### "+stampFirst) || !strings.Contains(string(got), "usage example") {
		t.Errorf("the second commit did not keep the first commit's heading:\n%s", got)
	}
}
