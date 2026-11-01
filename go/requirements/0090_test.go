package requirements

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test0090_1(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", localCheck("house-rule", passing))
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 0, allPassedWith(1))
	passedWith(t, out, "house-rule", "1")
}

func Test0090_2(t *testing.T) {
	t.Parallel()
	files := with(localCheck("markdown-no-bold-italic", passing), map[string]string{"README.md": readme + "\nSome **bold** text.\n"})
	out, code := fitness(t, example(t, "happyRepo", files), nil)
	sees(t, out, code, 0, "fitness: local checks replace built-in markdown-no-bold-italic", allPassed())
	if strings.Contains(out, "disallowed **bold**") {
		t.Errorf("the bundled markdown-no-bold-italic ran beside its local replacement:\n%s", out)
	}
	replacing := with(localCheck("markdown-no-bold-italic", passing), map[string]string{".fitnessrc.json": `{"policy": "external", "localChecks": "checks"}` + "\n"})
	out, code = boldPR(t, example(t, "happyRepo", replacing))
	sees(t, out, code, 0, "fitness: local checks replace built-in markdown-no-bold-italic", "Passed.")
}

func Test0090_3(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", localCheck("markdown-no-bold-italic", passing))
	out, code := fitness(t, repo, nil, "--all")
	sees(t, out, code, 1, "local check markdown-no-bold-italic replaces a built-in check, and org policy runs the built-in checks as the shared bar")
}

func Test0090_4(t *testing.T) {
	t.Parallel()
	for setting, want := range map[string]string{
		`{"localChecks": "checks"}`:             `fitness setup: .fitnessrc.json localChecks directory "checks" does not exist`,
		`{"localChecks": "../checks"}`:          `.fitnessrc.json localChecks "../checks" must be a path inside the repo`,
		`{"disabledChecks": ["no-such-check"]}`: `disabledChecks: unknown or empty check "no-such-check"`,
	} {
		repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": setting + "\n"})
		out, code := fitness(t, repo, nil)
		sees(t, out, code, 1, want)
	}
	out, code := fitness(t, example(t, "happyRepo", localCheck("boxes", passing)), nil)
	sees(t, out, code, 1, "local check boxes shares a fitness subcommand's name")
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"localChecks": "checks"}` + "\n"})
	mustDo(t, os.Symlink(t.TempDir(), filepath.Join(repo, "checks")))
	out, code = fitness(t, repo, nil)
	sees(t, out, code, 1, `.fitnessrc.json localChecks "checks" must be a path inside the repo`)
}

func Test0090_5(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", localCheck("house-rule", passing))
	mustDo(t, os.Chmod(filepath.Join(repo, "checks", "fitness-check-house-rule"), 0o644))
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 1, "local check checks/fitness-check-house-rule is not an executable file; run chmod +x checks/fitness-check-house-rule")
}

func Test0090_6(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", localCheck("house-rule", passing))
	out, code := fitness(t, repo, nil, "house-rule")
	sees(t, out, code, 0, "All 1 checks passed")
	passedWith(t, out, "house-rule", "1")
}

func Test0090_7(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{
		".fitnessrc.json": disabling,
		"README.md":       readme + "\nSome **bold** text.\n",
	})
	out, code := fitness(t, repo, nil)
	sees(t, out, code, 0, allPassedWith(-1))
	if row(out, "markdown-no-bold-italic") != "" {
		t.Errorf("disabled markdown-no-bold-italic still ran:\n%s", out)
	}
	out, code = boldPR(t, example(t, "happyRepo", nil))
	sees(t, out, code, 1, `disallowed \*\*bold`)
	out, code = boldPR(t, example(t, "happyRepo", map[string]string{".fitnessrc.json": `{"policy": "external", "disabledChecks": ["markdown-no-bold-italic"]}` + "\n"}))
	sees(t, out, code, 0, "Passed.")
}

func Test0090_8(t *testing.T) {
	t.Parallel()
	repo := example(t, "happyRepo", map[string]string{".fitnessrc.json": disabling})
	out, code := fitness(t, repo, nil, "--all")
	sees(t, out, code, 1, "disabledChecks turns off markdown-no-bold-italic, and org policy runs every check as the shared bar")
}

// allPassedWith is the summary of an all-passing run with extra checks beyond
// the bundled catalog (negative when some are off).
func allPassedWith(extra int) string {
	return fmt.Sprintf("All %d checks passed", len(strings.Fields(catalogRows))+extra)
}

// disabling is a config that turns markdown-no-bold-italic off.
const disabling = `{"disabledChecks": ["markdown-no-bold-italic"]}` + "\n"

// passing is a local check's verdict that judged one file and found nothing.
const passing = `{"ok": true, "errors": [], "filesChecked": 1}`

// localCheck is a repo whose checks directory holds one local check, name,
// that follows the check protocol and prints verdict.
func localCheck(name, verdict string) map[string]string {
	return map[string]string{
		".fitnessrc.json":              `{"localChecks": "checks"}` + "\n",
		"checks/fitness-check-" + name: "#!/bin/sh\nif [ \"$1\" = --describe ]; then\n  echo '{\"name\": \"" + name + "\"}'\n  exit 0\nfi\necho '" + verdict + "'\n",
	}
}

// boldPR runs pr-check in repo on a PR whose description uses bold, closing
// a Plan that is done.
func boldPR(t *testing.T, repo string) (string, int) {
	t.Helper()
	body := strings.Replace(pr0014(11), "This change is a test fixture.", "This change is a **bold** test fixture.", 1)
	plan := issue0014(t, 11, plan0014("- [x] Write the tests."), "Plan")
	rp := standIn(t, map[string][]response{"gh": {plan, noChangelog0014}})
	pr := map[string]any{"number": 1, "title": "test(app): add one file", "body": body}
	return fitness(t, repo, append(rp.env, event0014(t, "pull_request", map[string]any{"action": "opened", "pull_request": pr})...), "pr-check")
}
