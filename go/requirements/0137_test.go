package requirements

import (
	"path/filepath"
	"strings"
	"testing"
)

// inComments writes one comment, given as its lines of text, into a Go,
// a Swift and a TypeScript file. Each file uses its own marker. The comment
// starts on line 3 of each file.
func inComments(lines ...string) map[string]string {
	mark := func(marker, code string, lines []string) string {
		var b strings.Builder
		for _, l := range lines {
			b.WriteString(strings.TrimRight(marker+" "+l, " ") + "\n")
		}
		return b.String() + code
	}
	return map[string]string{
		"app/main.go":   "package main\n\n" + mark("//", "func main() {}\n", lines),
		"app/App.swift": "import Foundation\n\n" + mark("///", "func run() {}\n", lines),
		"app/index.ts":  "export {};\n\n" + mark("//", "export const x = 1;\n", lines),
	}
}

// atEachFile are the want lines naming line 3 of each file inComments writes,
// followed by message.
func atEachFile(message string) []string {
	return []string{"app/App.swift:3: " + message, "app/index.ts:3: " + message, "app/main.go:3: " + message}
}

// hardToRead is a comment of 104 words, each sentence within its limit,
// whose long words read past every alarm band.
var hardToRead = strings.Repeat("Configuration interdependencies necessitate comprehensive documentation regarding initialization sequencing, synchronization guarantees, and serialization compatibility. ", 8)

func Test0137_1(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, inComments(strings.TrimSpace(longSentence)))
	sees(t, out, code, 1, atEachFile("a sentence has 30 words (max 23)")...)
	out, code = proseBudget(t, inComments("One. Two. Three. Four. Five."))
	sees(t, out, code, 1, atEachFile("a paragraph has 5 sentences (max 4)")...)
}

func Test0137_2(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", inComments(hardToRead)), nil, "text-readability")
	sees(t, out, code, 1, atEachFile("readability alarm on 104 prose words")...)
}

func Test0137_3(t *testing.T) {
	t.Parallel()
	out, code := fitness(t, example(t, "happyRepo", inComments("It's not a cache. It's the source of truth.")), nil, "no-contrastive-reframing")
	sees(t, out, code, 1, atEachFile(`rewrite as a direct statement, not the "not X, it's Y" pattern`)...)
}

func Test0137_4(t *testing.T) {
	t.Parallel()
	words := strings.TrimSpace(longSentence)
	files := inComments("Retries once, since the first call warms the cache.")
	files["app/main.go"] = "//go:build " + strings.Repeat("linux && ", 29) + "darwin\n\n" + files["app/main.go"]
	files["app/build.go"] = "package main\n\n// cspell:ignore " + words + "\n\n// Runs the build:\n//\n//\t" + words + "\nfunc build() {}\n"
	files["app/Build.swift"] = "// cspell:ignore " + words + "\n/// " + strings.Repeat("One short paragraph.\n///\n/// ", 7) + "The end.\nfunc build() {}\n"
	files["app/build.ts"] = "// cspell:ignore " + words + "\n// Runs:\n//\n//     " + words + "\nexport const y = 2;\n"
	out, code := fitness(t, example(t, "happyRepo", files), nil,
		"--policy=external", "--checks=prose-budget,text-readability,no-contrastive-reframing")
	sees(t, out, code, 0, "All 3 checks passed")
}

func Test0137_5(t *testing.T) {
	t.Parallel()
	long := strings.TrimSpace(longSentence)
	repo := example(t, "happyRepo", map[string]string{
		"app/run main.go": "package main\n\n// " + long + "\nfunc old() {}\n\n// Runs.\nfunc main() {}\n",
	})
	write(t, repo, map[string]string{
		"app/run main.go": "package main\n\n// " + long + "\nfunc old() {}\n\n// Runs " + long + "\nfunc main() {}\n",
	})
	git(t, repo, "add", "-A")
	out, code := fitness(t, repo, nil, "prose-budget")
	sees(t, out, code, 1, "app/run main.go:6: a sentence has 31 words (max 23)")
	if strings.Contains(out, "app/run main.go:3:") {
		t.Errorf("the hook judged a comment the change left alone:\n%s", out)
	}
	out, code = fitness(t, repo, nil, "--all", "prose-budget")
	sees(t, out, code, 1, "app/run main.go:3: a sentence has 30 words (max 23)", "app/run main.go:6: a sentence has 31 words (max 23)")
}

func Test0137_6(t *testing.T) {
	t.Parallel()
	installer(t)
	out, code := fitness(t, filepath.Join("..", ".."), nil, "--policy=external", "--checks=prose-budget,text-readability,no-contrastive-reframing", "--dir=go/cmd")
	sees(t, out, code, 0, "All 3 checks passed")
}

func Test0137_7(t *testing.T) {
	t.Parallel()
	installer(t)
	out, code := fitness(t, filepath.Join("..", ".."), nil, "explain", "go-complexity")
	sees(t, out, code, 0, "Enforces a cyclomatic complexity ceiling on Go functions, mirroring the house eslint rule (`complexity: max 5`) that gates JavaScript and TypeScript.")
}

func Test0137_8(t *testing.T) {
	t.Parallel()
	installer(t)
	out, code := fitness(t, filepath.Join("..", ".."), nil, "explain", "mermaid-diagram-prose")
	sees(t, out, code, 0, "beyond a callout (a number or a letter) in a diagram paired with a callout table")
}
