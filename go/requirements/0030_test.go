package requirements

import "testing"

// gofmtCheck runs `fitness-install -- gofmt` on happyRepo with files written
// over it and env added.
func gofmtCheck(t *testing.T, env []string, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, "gofmt")
}

// gofmtTidy is a Go file gofmt leaves as it is; gofmtMessy is one it rewrites.
const (
	gofmtTidy  = "package m\n\nfunc F() int { return 1 }\n"
	gofmtMessy = "package m\nfunc   G( ) int {return 2}\n"
)

func Test0030_1(t *testing.T) {
	t.Parallel()
	out, code := gofmtCheck(t, nil, map[string]string{"ok.go": gofmtTidy, "bad.go": gofmtMessy})
	sees(t, out, code, 1, "bad.go: not gofmt-formatted (run: gofmt -w bad.go)")
}

func Test0030_2(t *testing.T) {
	t.Parallel()
	out, code := gofmtCheck(t, nil, map[string]string{"ok.go": gofmtTidy})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0030_3(t *testing.T) {
	t.Parallel()
	out, code := gofmtCheck(t, nil, map[string]string{
		"ok.go":                 gofmtTidy,
		"x/testdata/fixture.go": gofmtMessy,
		"vendor/lib/lib.go":     gofmtMessy,
	})
	sees(t, out, code, 0, "All 1 checks passed")
}

func Test0030_4(t *testing.T) {
	t.Parallel()
	out, code := gofmtCheck(t, nil, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "gofmt")
}

func Test0030_5(t *testing.T) {
	t.Parallel()
	out, code := gofmtCheck(t, []string{pathWithout(t, "go", "gofmt")}, map[string]string{"ok.go": gofmtTidy})
	sees(t, out, code, 1, "Go not installed: see https://go.dev/dl")
}

func Test0030_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{"ok.go": gofmtTidy})
	write(t, repo, map[string]string{"scratch.go": gofmtMessy})
	out, code := fitness(t, repo, nil, "gofmt")
	sees(t, out, code, 0, "All 1 checks passed")
}
