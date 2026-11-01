package requirements

import (
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
	files["app/Build.swift"] = "// cspell:ignore " + words + "\n/// " + strings.Repeat("One short paragraph here.\n///\n/// ", 7) + "The end is here.\nfunc build() {}\n"
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

// inBodies writes one comment, given as its lines of text, inside a
// function body of a Go, a Swift and a TypeScript file. The comment starts
// on line 4 of each file.
func inBodies(lines ...string) map[string]string {
	mark := func(indent, before, after string) string {
		var b strings.Builder
		b.WriteString(before)
		for _, l := range lines {
			b.WriteString(indent + "// " + l + "\n")
		}
		return b.String() + after
	}
	return map[string]string{
		"app/body.go":    mark("\t", "package main\n\nfunc body() {\n", "\tprintln()\n}\n"),
		"app/Body.swift": mark("    ", "import Foundation\n\nfunc body() {\n", "    print(1)\n}\n"),
		"app/body.ts":    mark("  ", "export {};\n\nexport function body(): void {\n", "  console.log(1);\n}\n"),
	}
}

// atEachBody are the want lines naming line 4 of each file inBodies
// writes, followed by message.
func atEachBody(message string) []string {
	return []string{"app/Body.swift:4: " + message, "app/body.go:4: " + message, "app/body.ts:4: " + message}
}

// split is the comment PR #746 passed with: one sentence, then a fragment
// that carries it on.
var split = []string{
	"run reads the whole model and every requirement doc unless the run is architecture.Unmoved.",
	"So a removed actor fails even when no requirement changed.",
}

// threeSentences is a comment of three whole sentences.
var threeSentences = []string{
	"Reads the cache first on every run.",
	"The cache may be cold after a deploy.",
	"One retry warms it for the next call.",
}

func Test0137_6(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, inComments(split...))
	sees(t, out, code, 1, atEachFile(`comment-fragment: a sentence starts with "So"; join it to the sentence before it or drop it`)...)
	out, code = proseBudget(t, inComments("Retries once on a cold cache.", "Cache warms first."))
	sees(t, out, code, 1, atEachFile("comment-fragment: a sentence has 3 words (min 4)")...)
	files := inBodies(threeSentences...)
	files["app/tail.go"] = "package main\n\nvar x = 1 // Holds one on every run. The value never changes. Tests read it directly.\n"
	out, code = proseBudget(t, files)
	want := "comment-one-statement: a comment in a function body or after code has 3 sentences (max 2); condense it to one statement"
	sees(t, out, code, 1, append(atEachBody(want), "app/tail.go:3: "+want)...)
	out, code = proseBudget(t, map[string]string{"app/fence.ts": "export {};\n\n// - Builds it:\n//   ```\n//   npm run build\n//   ```\n// So it ends.\nexport const y = 2;\n"})
	sees(t, out, code, 1, `app/fence.ts:3: comment-fragment: a sentence starts with "So"`)
}

func Test0137_7(t *testing.T) {
	t.Parallel()
	files := inComments(threeSentences...)
	for path, src := range inBodies("Retries the request once, since the first call after a deploy warms the cache that every later call reads from.") {
		files[path] = src
	}
	files["app/old.go"] = "package main\n\n// Deprecated: Use main.\nfunc old() {}\n\n// `helper` runs.\nfunc helper() {}\n"
	files["app/svc.ts"] = "export class Svc {\n  // " + strings.Join(threeSentences, "\n  // ") + "\n  run(x: number): number {\n    return x;\n  }\n}\n"
	files["app/list.go"] = "package main\n\n// Codes:\n//\n//   - 1: a check found problems, or the run could not start, such as a bad\n//     config.\n//\n// unwraps reads the ? or ! of a?.b, and \"- [ ]\" or \"1. [ ]\" items.\nfunc unwraps() {}\n"
	files["README.md"] = readme + "\nThe cache warms. So it works.\n"
	out, code := proseBudget(t, files)
	sees(t, out, code, 0)
}

func Test0137_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		"app/run.go": "package main\n\n// Runs old code.\nfunc old() {}\n\n// Runs the main loop until it ends.\nfunc main() {}\n",
	})
	write(t, repo, map[string]string{
		"app/run.go": "package main\n\n// Runs old code.\nfunc old() {}\n\n// Runs the main loop. So it ends.\nfunc main() {}\n",
	})
	git(t, repo, "add", "-A")
	out, code := fitness(t, repo, nil, "prose-budget")
	sees(t, out, code, 1, `app/run.go:6: comment-fragment: a sentence starts with "So"`)
	if strings.Contains(out, "app/run.go:3:") {
		t.Errorf("the hook judged a comment the change left alone:\n%s", out)
	}
	out, code = fitness(t, repo, nil, "--all", "prose-budget")
	sees(t, out, code, 1, "app/run.go:3: comment-fragment: a sentence has 3 words (min 4)", `app/run.go:6: comment-fragment: a sentence starts with "So"`)
}
