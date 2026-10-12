package requirements

import "testing"

func Test0071_1(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": ""}))
	sees(t, out, code, 1, "0001.1 has no test; name exactly one test Test0001_1")
}

func Test0071_2(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"more/more_test.go": testFile("more", "Test0001_1")}))
	sees(t, out, code, 1, "0001.1 is owned by 2 tests")
}

func Test0071_3(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "Test0001_2")}))
	sees(t, out, code, 1, "Test0001_2 names 0001.2, which no requirement defines")
}

func Test0071_4(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")}))
	sees(t, out, code, 1, "TestAddsAgain proves no requirement")
}

func Test0071_5(t *testing.T) {
	t.Parallel()
	out, code := judged(t, with(goRepo(), map[string]string{"docs/requirements/0001-adds-numbers.md": ""}))
	sees(t, out, code, 1, "docs/requirements/ has no requirement docs")
}

func Test0071_6(t *testing.T) {
	t.Parallel()
	files := with(goRepo(), map[string]string{"adder_test.go": testFile("adder", "Test0001_1", "TestAddsAgain")})
	unchangedTestFails(t, files, "TestAddsAgain proves no requirement")
}
