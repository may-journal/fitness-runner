package requirements

import (
	"regexp"
	"strings"
	"testing"
)

// times0166 matches a duration as the results table and summary print it.
var times0166 = regexp.MustCompile(`[0-9]+(\.[0-9]+)?(ms|s)\b`)

// results0166 is the results table and summary of a run, line by line, with
// times and the padding they set blanked.
func results0166(out string) string {
	start := strings.Index(out, "┌")
	end := strings.Index(out, "Slowest:")
	if start < 0 || end < start {
		return "no results table in:\n" + out
	}
	lines := strings.Split(times0166.ReplaceAllString(out[start:end], ""), "\n")
	for i, line := range lines {
		// A shorter time pads its cell, so runs of spaces count as one.
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.Join(lines, "\n")
}

func Test0166_1(t *testing.T) {
	t.Parallel()
	checks := "--checks=markdown-links,markdown-no-bold-italic,markdown-filename-kebab-case"
	repo := pinned0164(t, `{"markdown-links": "v1.0.0", "markdown-no-bold-italic": "v1.0.0", "markdown-filename-kebab-case": "v1.0.0"}`)
	commit(t, repo, map[string]string{"guide.md": "# Guide\n\nSee [the steps](missing.md) and **this**.\n"}, "docs(guide): add a guide")
	env := tags0164(t)
	builtIn, code := fiRun(t, repo, env, "--", "--policy=external", "--all", checks)
	sees(t, builtIn, code, 1, "missing.md", "disallowed **bold**")
	clean, code := fiRun(t, repo, env, "--with-checks", "--", "--no-builtins", "--policy=external", "--all", checks)
	sees(t, clean, code, 1, "installed 3 checks in ")
	if got, want := results0166(clean), results0166(builtIn); got != want {
		t.Errorf("the clean install must judge as the built-ins do\nclean:\n%s\nbuilt-in:\n%s", got, want)
	}
}
