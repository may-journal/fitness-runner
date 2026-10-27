// cspell:ignore mvnw gradlew

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

func Test0021_5(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{"cspell.json": `{"ignorePaths":["docs"]}` + "\n"})
	sees(t, out, code, 1, "cspell.json sets ignorePaths; cspell checks every tracked file, so remove ignorePaths and fix the findings instead")
}

func Test0021_11(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		".gitattributes":       "*.gen linguist-generated\n",
		"deps/pkg.gen":         cspellSharedWord + "\n",
		"blob.bin":             cspellBinaryWord + "\x00" + cspellBinaryOther,
		"package-lock.json":    `{"packages":{"node_modules/` + cspellTypo + `":{}}}` + "\n",
		"web/yarn.lock":        cspellRepoWord + "@1.0.0\n",
		"pnpm-lock.yaml":       cspellBodyTypo + ": 1.0.0\n",
		"go.sum":               "example.com/" + cspellBinaryOther + " v1.0.0 h1:abc=\n",
		"App/Package.resolved": `{"identity":"` + cspellBinaryWord + `"}` + "\n",
		"lib/gradle.lockfile":  "com.example:" + cspellTypo + ":1.0=compileClasspath\n",
		"mvnw":                 "url=" + cspellTypo + "\n",
		"app/gradlew.bat":      "set " + cspellRepoWord + "=1\n",
		".mvn/wrapper/MavenWrapperDownloader.java": "class Downloader { String " + cspellRepoWord + "; }\n",
		"gradle/wrapper/gradle-wrapper.properties": "distributionUrl=" + cspellBodyTypo + "\n",
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

func Test0021_13(t *testing.T) {
	t.Parallel()
	// 2,500 filler lines are 65,000 bytes, so the block opens just before
	// cspell's 64 KB chunk boundary and closes after it.
	filler := strings.Repeat("good words in a long file\n", 2500)
	long := "the " + cspellTypo + " fox\n" + filler +
		"cspell:disable\n" + filler[:2600] + "the " + cspellBodyTypo + " fox\ncspell:enable\n" +
		filler + "the " + cspellRepoWord + " fox\n"
	out, code := cspellRun(t, map[string]string{
		"notes.md": "good words\nthe " + cspellTypo + " fox\n",
		"long.md":  long,
	})
	first := "long.md:1:5 - Unknown word (" + cspellTypo + ")"
	last := "long.md:5105:5 - Unknown word (" + cspellRepoWord + ")"
	sees(t, out, code, 1, "notes.md:2:5 - Unknown word ("+cspellTypo+")", first, last)
	if strings.Contains(out, cspellBodyTypo) || strings.Index(out, first) > strings.Index(out, last) {
		t.Errorf("only the words outside the block may be named, in line order:\n%s", out)
	}
}

func Test0021_8(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{"body.md": "a " + cspellBodyTypo + " word\n"}, "--body-file", "body.md")
	sees(t, out, code, 1, "(description):1:3 - Unknown word ("+cspellBodyTypo+")")
}

func Test0021_12(t *testing.T) {
	t.Parallel()
	out, code := cspellRun(t, map[string]string{
		cspellInstalled: `{"words":["fitnessrc","` + cspellSharedWord + `"]}` + "\n",
		"notes.md":      cspellSharedWord + " is a shared word\n",
	})
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = cspellRun(t, map[string]string{"notes.md": "mayjournal is a shared word, and addrinfo a language one\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	out, code = cspellRun(t, map[string]string{"body.md": "sync `fastlane/Appfile`, `fastlane/Fastfile`, `fastlane/Gymfile`\n"}, "--body-file", "body.md")
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0021_10(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"cspell.json": `{"words":["fitnessrc"],"flagWords":["green apple"]}` + "\n",
		"notes.md":    "a Green  Apple and green" + "apple\n",
		"body.md":     "no green\napple here\n",
	}
	out, code := cspellRun(t, files)
	sees(t, out, code, 1, "notes.md:1:3 - Forbidden word (green apple)", "body.md:1:4 - Forbidden word (green apple)")
	if strings.Contains(out, "cspell.json:") {
		t.Errorf("the cspell.json listing flagWords must not be flagged:\n%s", out)
	}
	out, code = cspellRun(t, files, "--body-file", "body.md")
	sees(t, out, code, 1, "(description):1:4 - Forbidden word (green apple)")
}
