package requirements

import (
	"strings"
	"testing"
)

// inFunctions writes each Go, Swift and TypeScript body, its comment from line 4, inside a function.
func inFunctions(goBody, swiftBody, tsBody string) map[string]string {
	return map[string]string{
		"app/run.go":    "package main\n\nfunc run(a int) int {\n" + goBody + "\treturn a\n}\n",
		"app/Run.swift": "import Foundation\n\nfunc run(a: Int) -> Int {\n" + swiftBody + "    return a\n}\n",
		"app/run.ts":    "export {};\n\nexport function run(a: number): number {\n" + tsBody + "  return a;\n}\n",
	}
}

// atEachFunction names line 4 of each inFunctions file, followed by message.
func atEachFunction(message string) []string {
	return []string{"app/Run.swift:4: " + message, "app/run.go:4: " + message, "app/run.ts:4: " + message}
}

func Test0139_1(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, inFunctions(
		"\t// x := compute(a)\n\t// if x > 0 {\n\t// \treturn x\n\t// }\n",
		"    // let x = compute(a)\n    // print(x)\n",
		"  // const x = compute(a);\n  // console.log(x);\n"))
	sees(t, out, code, 1,
		"app/run.go:4: commented-out-code: 3 of 3 lines are code; delete it, since git keeps old code",
		"app/Run.swift:4: commented-out-code: 2 of 2 lines are code; delete it",
		"app/run.ts:4: commented-out-code: 2 of 2 lines are code; delete it")
}

func Test0139_2(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, inFunctions(
		"\ti := 0\n\t// increment i\n\ti++\n",
		"    var i = 0\n    // increment i\n    i += 1\n",
		"  let i = 0;\n  // increment i\n  i++;\n"))
	sees(t, out, code, 1,
		"app/run.go:5: restates-code: every word repeats the next code line; delete it or say why",
		"app/Run.swift:5: restates-code: every word repeats the next code line",
		"app/run.ts:5: restates-code: every word repeats the next code line")
}

func Test0139_3(t *testing.T) {
	t.Parallel()
	files := inComments("🚀 Launches the rocket once on start.")
	files["app/banner.go"] = "package main\n\n// ==========\nvar x = 1\n"
	out, code := proseBudget(t, files)
	sees(t, out, code, 1, append(atEachFile(`decoration: a comment holds the emoji "🚀"; delete it`),
		`app/banner.go:3: decoration: a comment holds the banner "=========="; delete it`)...)
}

func Test0139_4(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, inComments("Previously read the cache on every run."))
	sees(t, out, code, 1, atEachFile(`change-narration: "Previously" narrates a change; say what holds today, since git keeps history`)...)
}

func Test0139_5(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, inComments("Basically retries once on a cold cache."))
	sees(t, out, code, 1, atEachFile(`hedging: "Basically" hedges; drop the word or state the fact`)...)
}

func Test0139_6(t *testing.T) {
	t.Parallel()
	files := inComments("Retries just once, since the first call warms the cache.")
	files["app/state.go"] = "package main\n\n// Maps each state → its next state on every tick.\nvar next = map[int]int{}\n"
	files["app/expiry.ts"] = "export {};\n\n// The cache no longer expires once it holds a value.\nexport const keep = true;\n"
	files["app/store.go"] = "package main\n\n// Returns the previously stored value.\nfunc stored() int { return 0 }\n"
	files["README.md"] = readme + "\nIt basically worked previously. 🚀\n\n-----\n"
	out, code := proseBudget(t, files)
	sees(t, out, code, 0)
}

func Test0139_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		"app/run.go": "package main\n\n// Basically runs the old code.\nfunc old() {}\n\n// Runs the main loop until it ends.\nfunc main() {}\n",
	})
	write(t, repo, map[string]string{
		"app/run.go": "package main\n\n// Basically runs the old code.\nfunc old() {}\n\n// Runs the main loop, simply until it ends.\nfunc main() {}\n",
	})
	git(t, repo, "add", "-A")
	out, code := fitness(t, repo, nil, "prose-budget")
	sees(t, out, code, 1, `app/run.go:6: hedging: "simply" hedges`)
	if strings.Contains(out, "app/run.go:3:") {
		t.Errorf("the hook judged a comment the change left alone:\n%s", out)
	}
	out, code = fitness(t, repo, nil, "--all", "prose-budget")
	sees(t, out, code, 1, `app/run.go:3: hedging: "Basically" hedges`, `app/run.go:6: hedging: "simply" hedges`)
}
