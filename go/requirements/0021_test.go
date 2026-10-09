package requirements

import (
	"strings"
	"testing"
)

// Misspellings are built from pieces so this file's own spell check passes.
var (
	cspellTypo        = "qu" + "ik"
	cspellRepoWord    = "bor" + "wn"
	cspellSharedWord  = "zz" + "zq" + "qqv"
	cspellBinaryWord  = "xq" + "zzt"
	cspellBinaryOther = "vb" + "nmm"
	cspellBodyTypo    = "mis" + "pe" + "led"
)

// cspellRun runs `fitness-install -- cspell <args>` on happyRepo with files
// written over it and committed.
func cspellRun(t *testing.T, files map[string]string, args ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, append([]string{"cspell"}, args...)...)
}

// cspellInstalled is where an installed shared package keeps its cspell.json.
const cspellInstalled = "node_modules/@mayjournal/fitness-shared/config/cspell.json"

func Test0021_1(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{"notes.md": "good words\nthe " + cspellTypo + " fox\n"})
	sees(t, out, code, 1, "notes.md:2:5 - Unknown word ("+cspellTypo+")")
}

func Test0021_2(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{
		"cspell.json":   `{"words":["fitnessrc","` + cspellRepoWord + `"]}` + "\n",
		cspellInstalled: `{"words":["fitnessrc","` + cspellSharedWord + `"]}` + "\n",
		"notes.md":      cspellRepoWord + " and " + cspellSharedWord + "\n",
	})
	sees(t, out, code, 1, "notes.md:1:11 - Unknown word ("+cspellSharedWord+")")
	if strings.Contains(out, "Unknown word ("+cspellRepoWord+")") {
		t.Errorf("the repo's own word must be accepted:\n%s", out)
	}
}

func Test0021_3(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{
		cspellInstalled: `{"words":["fitnessrc","` + cspellSharedWord + `"]}` + "\n",
		"notes.md":      cspellSharedWord + " is a shared word\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0021_4(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{"notes.md": "mayjournal is a shared word\n"})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0021_5(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{"cspell.json": `{"ignorePaths":["docs"]}` + "\n"})
	sees(t, out, code, 1, "cspell.json sets ignorePaths; cspell checks every tracked file, so remove ignorePaths and fix the findings instead")
}

func Test0021_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		".gitattributes": "*.lock linguist-generated\n",
		"deps/pkg.lock":  cspellSharedWord + "\n",
		"blob.bin":       cspellBinaryWord + "\x00" + cspellBinaryOther,
	})
	write(t, repo, map[string]string{"untracked.md": "the " + cspellRepoWord + " fox\n"})
	out, code := fitness(t, repo, nil, "cspell")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0021_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"old.md": "the " + cspellTypo + " fox\n", "gone.md": "clean words\n"})
	write(t, repo, map[string]string{"new.md": "the " + cspellRepoWord + " fox\n"})
	git(t, repo, "add", "new.md")
	git(t, repo, "rm", "-q", "gone.md")
	out, code := fitness(t, repo, nil, "cspell")
	sees(t, out, code, 1, "new.md:1:5 - Unknown word ("+cspellRepoWord+")")
	if strings.Contains(out, cspellTypo) || strings.Contains(out, "gone.md") {
		t.Errorf("only the staged file on disk may be checked:\n%s", out)
	}
}

func Test0021_8(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{"body.md": "a " + cspellBodyTypo + " word\n"}, "--body-file", "body.md")
	sees(t, out, code, 1, "(description):1:3 - Unknown word ("+cspellBodyTypo+")")
}
