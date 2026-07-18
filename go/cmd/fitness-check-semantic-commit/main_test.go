package main

import (
	"os"
	"os/exec"
	"testing"
)

const typesList = "feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"

func TestIsSemanticSubject(t *testing.T) {
	cases := []struct {
		name    string
		subject string
		want    bool
	}{
		{"feat with scope", "feat(api): add endpoint", true},
		{"fix with scope", "fix(deps): bump cspell", true},
		{"docs with scope", "docs(readme): update links", true},
		{"merge commit", "Merge branch 'x' into main", true},
		{"plain sentence", "Fix something", false},
		{"missing scope", "feat: no scope", false},
		{"unknown type", "invalid(scope): missing type", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSemanticSubject(tc.subject); got != tc.want {
				t.Fatalf("isSemanticSubject(%q) = %v, want %v", tc.subject, got, tc.want)
			}
		})
	}
}

func TestJudge(t *testing.T) {
	cases := []struct {
		name      string
		subject   string
		ok        bool
		wantError string
	}{
		{"semantic subject passes", "feat(api): add endpoint", true, ""},
		{"merge subject passes", "Merge branch 'feature' into main", true, ""},
		{"empty subject fails", "", false, msgEmpty},
		{"whitespace subject fails", "   ", false, msgEmpty},
		{"non-semantic subject fails with types", "wip stuff", false,
			`Commit message: "wip stuff" — use type(scope): description (types: ` + typesList + `)`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judge(tc.subject)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 1 {
				t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
			}
			if !tc.ok && res.Errors[0] != tc.wantError {
				t.Fatalf("error = %q, want %q", res.Errors[0], tc.wantError)
			}
		})
	}
}

// withMessage sets FITNESS_CTX_MESSAGE for the test; withoutMessage
// guarantees it is absent (t.Setenv registers the restore).
func withMessage(t *testing.T, msg string) {
	t.Helper()
	t.Setenv("FITNESS_CTX_MESSAGE", msg)
}

func withoutMessage(t *testing.T) {
	t.Helper()
	t.Setenv("FITNESS_CTX_MESSAGE", "")
	if err := os.Unsetenv("FITNESS_CTX_MESSAGE"); err != nil {
		t.Fatal(err)
	}
}

func TestRunValidatesProvidedMessage(t *testing.T) {
	withMessage(t, "feat(api): add endpoint")
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected pass, got %+v", res)
	}
}

func TestRunValidatesOnlyFirstLine(t *testing.T) {
	withMessage(t, "chore(scope): description\n\nMade-with: Cursor")
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected pass on first line, got %+v", res)
	}
}

func TestRunFailsOnEmptyProvidedMessage(t *testing.T) {
	// Present-but-empty must not fall back to git log (tri-state).
	withMessage(t, "")
	res, err := run(headRepo(t, "feat(api): add endpoint"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.Errors[0] != msgEmpty {
		t.Fatalf("expected msgEmpty failure, got %+v", res)
	}
}

func TestRunFallsBackToHeadMessage(t *testing.T) {
	withoutMessage(t)
	pass, err := run(headRepo(t, "feat(api): add endpoint\n\nBody"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !pass.Ok {
		t.Fatalf("expected pass from HEAD, got %+v", pass)
	}
	fail, err := run(headRepo(t, "Fix something\n\nBody"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if fail.Ok || fail.Errors[0] !=
		`Commit message: "Fix something" — use type(scope): description (types: `+typesList+`)` {
		t.Fatalf("expected format failure from HEAD, got %+v", fail)
	}
}

func TestRunFailsOutsideGitRepo(t *testing.T) {
	withoutMessage(t)
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.Errors[0] != msgEmpty {
		t.Fatalf("expected msgEmpty failure, got %+v", res)
	}
}

// headRepo creates a temp git repo whose HEAD commit carries msg.
func headRepo(t *testing.T, msg string) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	git("config", "commit.gpgsign", "false")
	git("commit", "-q", "--allow-empty", "-m", msg)
	return dir
}
