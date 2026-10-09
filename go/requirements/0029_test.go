package requirements

import (
	"regexp"
	"testing"
)

// goVetMod is a minimal go.mod for a Given's module.
const goVetMod = "module m\n\ngo 1.21\n"

// goVetBad is Go source the compiler accepts but go vet flags.
const goVetBad = "package m\n\nimport \"fmt\"\n\nfunc F() { fmt.Printf(\"%d\\n\", \"x\") }\n"

// goVetFinding is what go vet reports for goVetBad.
const goVetFinding = `m.go:5:24: fmt.Printf format %d has arg "x" of wrong type string`

// goVetModule returns the files of a module in dir holding src.
func goVetModule(dir, src string) map[string]string {
	return map[string]string{dir + "/go.mod": goVetMod, dir + "/m.go": src}
}

// goVet runs `fitness-install -- go-vet` in repo with env added.
func goVet(t *testing.T, repo string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, repo, env, "go-vet")
}

func Test0029_1(t *testing.T) {
	out, code := goVet(t, example(t, "happyRepo", goVetModule(".", goVetBad)))
	sees(t, out, code, 1, "✖ "+goVetFinding)
}

func Test0029_2(t *testing.T) {
	out, code := goVet(t, example(t, "happyRepo", goVetModule("api", goVetBad)))
	sees(t, out, code, 1, "✖ api: "+goVetFinding)
}

func Test0029_3(t *testing.T) {
	out, code := goVet(t, example(t, "happyRepo", goVetModule("api", "package m\n\nfunc F() int { return 1 }\n")))
	sees(t, out, code, 0, "All 1 checks passed")
	if !regexp.MustCompile(`│ passed\s+│ 1\s+│`).MatchString(row(out, "go-vet")) {
		t.Errorf("go-vet must pass counting 1 module, got row %q", row(out, "go-vet"))
	}
}

func Test0029_4(t *testing.T) {
	out, code := goVet(t, example(t, "happyRepo", map[string]string{"main.go": goVetBad}))
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "go-vet")
}

func Test0029_5(t *testing.T) {
	files := goVetModule("testdata/fixture", goVetBad)
	for name, body := range goVetModule("vendor/dep", goVetBad) {
		files[name] = body
	}
	out, code := goVet(t, example(t, "happyRepo", files))
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "go-vet")
}

func Test0029_6(t *testing.T) {
	repo := example(t, "happyRepo", goVetModule("api", goVetBad))
	write(t, repo, map[string]string{"README.md": readme + "\nMore text.\n"})
	git(t, repo, "add", "README.md")
	out, code := goVet(t, repo)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "go-vet")
}

func Test0029_7(t *testing.T) {
	out, code := goVet(t, example(t, "happyRepo", goVetModule("api", goVetBad)), "PATH=/usr/bin:/bin")
	sees(t, out, code, 1, "✖ Go not installed: see https://go.dev/dl")
}
