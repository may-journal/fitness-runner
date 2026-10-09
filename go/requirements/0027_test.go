package requirements

import "testing"

// goTest runs `fitness-install -- go-test` on happyRepo with files written
// over it and env added.
func goTest(t *testing.T, files map[string]string, env ...string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), env, "go-test")
}

// goTestModule is a Go module in dir whose test file holds body.
func goTestModule(dir, body string) map[string]string {
	return map[string]string{
		dir + "go.mod":    "module example.com/m\n\ngo 1.22\n",
		dir + "m_test.go": "package m\n\nimport \"testing\"\n\n" + body,
	}
}

// goTestBroken is a test that fails with the message "boom" on line 5.
const goTestBroken = "func TestBroken(t *testing.T) { t.Error(\"boom\") }\n"

func Test0027_1(t *testing.T) {
	t.Parallel()
	out, code := goTest(t, goTestModule("", goTestBroken))
	sees(t, out, code, 1, "--- FAIL: TestBroken", "m_test.go:5: boom")
}

func Test0027_2(t *testing.T) {
	t.Parallel()
	out, code := goTest(t, goTestModule("", "func TestFine(t *testing.T) {}\n"))
	sees(t, out, code, 0, "All 1 checks passed · 1 files scanned")
}

func Test0027_3(t *testing.T) {
	t.Parallel()
	out, code := goTest(t, nil)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "go-test")
}

func Test0027_4(t *testing.T) {
	t.Parallel()
	files := with(goTestModule("testdata/a/", goTestBroken), goTestModule("vendor/b/", goTestBroken))
	out, code := goTest(t, files)
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "go-test")
}

func Test0027_5(t *testing.T) {
	t.Parallel()
	out, code := goTest(t, goTestModule("svc/", goTestBroken))
	sees(t, out, code, 1, "svc: --- FAIL: TestBroken", "svc: m_test.go:5: boom")
}

func Test0027_6(t *testing.T) {
	t.Parallel()
	out, code := goTest(t, goTestModule("", "func TestBroken(t *testing.T) { undefinedThing() }\n"))
	sees(t, out, code, 1, "m_test.go:5:33: undefined: undefinedThing")
}

func Test0027_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", goTestModule("", goTestBroken))
	write(t, repo, map[string]string{"README.md": readme + "\nOne more line.\n"})
	git(t, repo, "add", "README.md")
	out, code := fitness(t, repo, nil, "go-test")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "go-test")
}

func Test0027_8(t *testing.T) {
	t.Parallel()
	out, code := goTest(t, goTestModule("", "func TestFine(t *testing.T) {}\n"), pathWithout(t, "go"))
	sees(t, out, code, 1, "Go not installed: see https://go.dev/dl")
}
