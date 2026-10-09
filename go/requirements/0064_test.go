package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// notesHead opens a changelog in the house layout.
const notesHead = "---\nrelatedConfigurations: ['.fitnessrc.json']\n---\n\n# Changelog\n\n## Changes\n\n"

// notesTail is what follows the new release: a timestamped entry and an
// older release, both already in the house format.
const notesTail = "### 2026.10.09.1500\n\n- Fix: one.\n- Fix: two.\n- Fix: three.\n\n## 1.1.0 (2026-10-09)\n\n- Feat: older release.\n"

// notesPlease is a 1.1.1 release as Release Please writes it, under the
// given subheadings, each holding one bullet with a scope and links.
func notesPlease(subheadings ...string) string {
	s := "## [1.1.1](https://github.com/o/r/compare/v1.1.0...v1.1.1) (2026-10-10)\n\n"
	for _, h := range subheadings {
		s += "\n### " + h + "\n\n* **install:** start faster ([#239](https://github.com/o/r/issues/239)) ([c9f4729](https://github.com/o/r/commit/c9f4729))\n\n"
	}
	return s
}

// notesRun writes changelog into a release repo, runs `fitness-release
// house-changelog`, and returns the output, exit code, and the file after.
func notesRun(t *testing.T, changelog string) (string, int, string) {
	t.Helper()
	repo := releaseToolRepo(t, map[string]string{"CHANGELOG.md": changelog, "version.txt": "1.1.1\n"})
	out, code := releaseToolRun(t, repo, nil, "house-changelog")
	data, err := os.ReadFile(filepath.Join(repo, "CHANGELOG.md"))
	mustDo(t, err)
	return out, code, string(data)
}

func Test0064_1(t *testing.T) {
	t.Parallel()
	_, code, after := notesRun(t, notesHead+notesPlease("Bug Fixes")+notesTail)
	want := "## 1.1.1 (2026-10-10)\n\n- Fix: install: start faster ([#239](https://github.com/o/r/issues/239)) ([c9f4729](https://github.com/o/r/commit/c9f4729))\n\n### 2026.10.09.1500"
	if code != 0 || !strings.Contains(after, want) {
		t.Errorf("exit %d; want the release rewritten as\n%s\ngot\n%s", code, want, after)
	}
}

func Test0064_2(t *testing.T) {
	t.Parallel()
	_, code, after := notesRun(t, notesHead+notesPlease("Features", "Bug Fixes", "Performance Improvements")+notesTail)
	for _, kind := range []string{"- Feat: install:", "- Fix: install:", "- Perf: install:"} {
		if code != 0 || !strings.Contains(after, kind) {
			t.Errorf("exit %d; missing %q in\n%s", code, kind, after)
		}
	}
}

func Test0064_3(t *testing.T) {
	t.Parallel()
	_, code, after := notesRun(t, notesHead+notesPlease("Bug Fixes")+notesTail)
	if code != 0 || !strings.HasPrefix(after, notesHead) || !strings.HasSuffix(after, notesTail) {
		t.Errorf("exit %d; the lines around the release changed:\n%s", code, after)
	}
}

func Test0064_4(t *testing.T) {
	t.Parallel()
	house := notesHead + "## 1.1.1 (2026-10-10)\n\n- Perf: install: start faster.\n\n" + notesTail
	_, code, after := notesRun(t, house)
	if code != 0 || after != house {
		t.Errorf("exit %d; a house-format changelog changed:\n%s", code, after)
	}
}

func Test0064_5(t *testing.T) {
	t.Parallel()
	out, code, _ := notesRun(t, notesHead+notesPlease("Shiny Things")+notesTail)
	sees(t, out, code, 1, `unknown release subheading "Shiny Things"`)
}

func Test0064_6(t *testing.T) {
	t.Parallel()
	repo := releaseToolRepo(t, map[string]string{"CHANGELOG.md": notesHead + notesPlease("Features", "Bug Fixes") + notesTail})
	if out, code := releaseToolRun(t, repo, nil, "house-changelog"); code != 0 {
		t.Fatalf("house-changelog: %s", out)
	}
	out, code := fitness(t, repo, nil, "--policy=external", "--checks=changelog,changelog-bullets", "--all")
	sees(t, out, code, 0, "All 2 checks passed")
}
