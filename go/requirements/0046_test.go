package requirements

import "testing"

// noPlansDir runs `fitness-install -- no-plans-dir` on happyRepo with files
// written over it.
func noPlansDir(t *testing.T, files map[string]string) (string, int) {
	t.Helper()
	return fitness(t, example(t, "happyRepo", files), nil, "no-plans-dir")
}

// noPlansDirFix is the guidance printed after each offending file.
const noPlansDirFix = ": plans live as `Plan`-labeled GitHub Issues, not files — open a Plan Issue instead of adding to docs/plans/"

func Test0046_1(t *testing.T) {
	out, code := noPlansDir(t, map[string]string{"docs/plans/03-publish.md": "# Publish\n"})
	sees(t, out, code, 1, "docs/plans/03-publish.md"+noPlansDirFix)
}

func Test0046_2(t *testing.T) {
	out, code := noPlansDir(t, map[string]string{"docs/plans/archive/old.md": "# Old\n"})
	sees(t, out, code, 1, "docs/plans/archive/old.md"+noPlansDirFix)
}

func Test0046_3(t *testing.T) {
	out, code := noPlansDir(t, map[string]string{"docs/notes.md": "# Notes\n"})
	sees(t, out, code, 0, "All 1 checks passed")
	passedWithNoFiles(t, out, "no-plans-dir")
}
