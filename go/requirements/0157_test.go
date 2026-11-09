package requirements

import "testing"

func Test0157_1(t *testing.T) {
	t.Parallel()
	out, code := proseBudget(t, inFunctions(
		"\t// Hands the number back to the caller without any change at all.\n\n\t// Callers rely on getting their own number back from this function.\n",
		"    // Hands the number back to the caller without any change at all.\n\n    // Callers rely on getting their own number back from this function.\n",
		"  // Hands the number back to the caller without any change at all.\n\n  // Callers rely on getting their own number back from this function.\n"))
	sees(t, out, code, 1,
		"app/run.go:3: overlong: a function body holds 23 comment words in 2 comments for 2 code tokens (max 3 words per token); keep the comments that say why and cut the rest",
		"app/Run.swift:3: overlong: a function body holds 23 comment words in 2 comments for 2 code tokens",
		"app/run.ts:3: overlong: a function body holds 23 comment words in 2 comments for 3 code tokens")
}

func Test0157_2(t *testing.T) {
	t.Parallel()
	files := inFunctions(
		"\t// Hands the number back unchanged, since callers compare it with the value they passed in.\n",
		"    var b = a\n    // Only the copy changes below this line.\n    b += 1\n    // The copy is dropped once the caller has its number.\n    _ = b\n",
		"  let b = a;\n  // Only the copy changes below this line.\n  b++;\n  // The copy is dropped once the caller has its number.\n  void b;\n")
	files["app/swap.go"] = "package main\n\nfunc swap(a, b int) (int, int) {\n\t// Swaps the pair, as this call shows:\n" +
		"\t//     first, second := swap(the value the caller passed first, the value the caller passed second)\n" +
		"\t//     first, second = swap(the value that came back first, the value that came back second)\n" +
		"\tt := a\n\n\t// The copy keeps a safe while b moves.\n\ta = b\n\treturn a, t\n}\n"
	files["app/greet.sh"] = "#!/bin/sh\ngreet() {\n# echo hi\necho hi\n}\n"
	files["README.md"] = readme + "\nAn over-commented example:\n\n```go\nfunc run(a int) int {\n" +
		"\t// Hands the number back to the caller without any change at all.\n\n" +
		"\t// Callers rely on getting their own number back from this function.\n\treturn a\n}\n```\n"
	out, code := proseBudget(t, files)
	sees(t, out, code, 0)
}

func Test0157_3(t *testing.T) {
	t.Parallel()
	long := "\t// Hands the number back to the caller without any change at all.\n\n\t// Callers rely on getting their own number back from this function.\n"
	files := func(cfg string) map[string]string {
		f := map[string]string{"app/run.go": inFunctions(long, long, long)["app/run.go"]}
		if cfg != "" {
			f[".fitnessrc.json"] = cfg + "\n"
		}
		return f
	}
	out, code := proseBudget(t, files(""))
	sees(t, out, code, 1, "overlong", "(max 3 words per token)")
	out, code = proseBudget(t, files(`{"prose": {"comments": {"maxWordsPerToken": 20}}}`))
	sees(t, out, code, 0)
	out, code = proseBudget(t, files(`{"prose": {"comments": {"maxWordsPerToken": 2}}}`))
	sees(t, out, code, 1, "(max 2 words per token)")
	out, code = proseBudget(t, files(`{"prose": {"comments": {"minBodyComments": 3}}}`))
	sees(t, out, code, 0)
}
