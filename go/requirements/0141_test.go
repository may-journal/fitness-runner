package requirements

import "testing"

func Test0141_1(t *testing.T) {
	t.Parallel()
	files := inComments("Note that the cache warms once on start.")
	files["app/plan.go"] = "package main\n\n// Retries once on a miss. Let's keep the cache warm.\nfunc plan() {}\n"
	files["app/Seen.swift"] = "import Foundation\n\n/// As you can see, the cache warms on start.\nfunc seen() {}\n"
	files["app/steps.ts"] = "export {};\n\n// First we read the cache, then the disk.\nexport const steps = 2;\n"
	out, code := proseBudget(t, files)
	sees(t, out, code, 1, append(atEachFile(`tutorial-tone: "Note that" opens a sentence like a tutorial; state the rule plainly`),
		`app/plan.go:3: tutorial-tone: "Let's" opens a sentence`,
		`app/Seen.swift:3: tutorial-tone: "As you can see" opens a sentence`,
		`app/steps.ts:3: tutorial-tone: "First we" opens a sentence`)...)
}

func Test0141_2(t *testing.T) {
	t.Parallel()
	files := inComments("Writes a note that names the cache.")
	files["app/plan.go"] = "package main\n\n// Lets the caller retry once on a miss.\nfunc plan() {}\n"
	files["app/greet.ts"] = "export {};\n\n// Prints \"Let's go\" when the run starts.\nexport const greet = 1;\n"
	files["README.md"] = readme + "\nNote that it works. Let's see.\n"
	out, code := proseBudget(t, files)
	sees(t, out, code, 0)
}
