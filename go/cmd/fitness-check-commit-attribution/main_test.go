package main

import (
	"os"
	"os/exec"
	"testing"
)

const validMessage = "feat(api): add endpoint\n\nBody\n\nAI-Tools: Claude Code\nAI-Models: Opus 4.8"

const (
	errMissingModels = `commit message missing "AI-Models:" trailer`
	errMissingTools  = `commit message missing "AI-Tools:" trailer`
)

func TestJudge(t *testing.T) {
	cases := []struct {
		name       string
		message    string
		ok         bool
		wantErrors []string
	}{
		{"both trailers pass", validMessage, true, nil},
		{"missing AI-Tools fails", "feat(api): add endpoint\n\nAI-Models: Opus 4.8", false,
			[]string{errMissingTools}},
		{"missing AI-Models fails", "feat(api): add endpoint\n\nAI-Tools: Claude Code", false,
			[]string{errMissingModels}},
		{"neither trailer names both, AI-Models first", "feat(api): add endpoint\n\nJust a body", false,
			[]string{errMissingModels, errMissingTools}},
		{"merge commit exempt", "Merge branch 'feature' into main", true, nil},
		{"revert commit exempt", `Revert "feat(api): add endpoint"`, true, nil},
		{"indented Merge is not exempt", " Merge branch 'feature'", false,
			[]string{errMissingModels, errMissingTools}},
		{"empty message fails with the empty-message error", "", false, []string{msgEmpty}},
		{"blank subject fails even with trailers below", "   \nAI-Tools: x\nAI-Models: y", false,
			[]string{msgEmpty}},
		{"trailer with empty value counts as missing", "feat(api): add endpoint\n\nAI-Tools:\nAI-Models:   ", false,
			[]string{errMissingModels, errMissingTools}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := judge(tc.message)
			if res.Ok != tc.ok {
				t.Fatalf("ok = %v, want %v (errors: %v)", res.Ok, tc.ok, res.Errors)
			}
			if res.FilesChecked != 1 {
				t.Fatalf("filesChecked = %d, want 1", res.FilesChecked)
			}
			if len(res.Errors) != len(tc.wantErrors) {
				t.Fatalf("errors = %v, want %v", res.Errors, tc.wantErrors)
			}
			for i, want := range tc.wantErrors {
				if res.Errors[i] != want {
					t.Fatalf("errors[%d] = %q, want %q", i, res.Errors[i], want)
				}
			}
		})
	}
}

func TestHasTrailer(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    bool
	}{
		{"non-empty value on its own line", "subject\n\nAI-Tools: Claude Code", true},
		{"no value", "subject\n\nAI-Tools:", false},
		{"whitespace-only value", "subject\n\nAI-Tools:   ", false},
		{"absent", "subject\n\nno trailer here", false},
		{"mid-line mention does not count", "subject\n\nsee AI-Tools: Claude Code", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasTrailer(tc.message, "AI-Tools"); got != tc.want {
				t.Fatalf("hasTrailer(%q) = %v, want %v", tc.message, got, tc.want)
			}
		})
	}
}

// initRepo creates a temp git repo whose HEAD carries the given message.
func initRepo(t *testing.T, message string) string {
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
	git("commit", "--allow-empty", "-q", "-m", message)
	return dir
}

// unsetCtxMessage clears FITNESS_CTX_MESSAGE for the test, restoring after.
func unsetCtxMessage(t *testing.T) {
	t.Helper()
	t.Setenv("FITNESS_CTX_MESSAGE", "")
	if err := os.Unsetenv("FITNESS_CTX_MESSAGE"); err != nil {
		t.Fatal(err)
	}
}

func TestRunFallsBackToHeadMessage(t *testing.T) {
	unsetCtxMessage(t)
	res, err := run(initRepo(t, validMessage), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok || res.FilesChecked != 1 {
		t.Fatalf("expected pass on HEAD with trailers, got %+v", res)
	}
	res, err = run(initRepo(t, "feat(api): add endpoint\n\nno trailers"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || len(res.Errors) != 2 {
		t.Fatalf("expected two trailer errors from HEAD, got %+v", res)
	}
}

func TestRunOutsideGitRepoFails(t *testing.T) {
	unsetCtxMessage(t)
	res, err := run(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.Errors[0] != msgEmpty {
		t.Fatalf("expected empty-message failure, got %+v", res)
	}
}

func TestRunUsesContextMessage(t *testing.T) {
	dir := initRepo(t, "feat(api): add endpoint\n\nno trailers")
	t.Setenv("FITNESS_CTX_MESSAGE", validMessage)
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ok {
		t.Fatalf("expected context message to pass, got %+v", res)
	}
}

func TestRunEmptyContextMessageDoesNotFallBack(t *testing.T) {
	// HEAD would pass; a present-but-empty context message must still fail.
	dir := initRepo(t, validMessage)
	t.Setenv("FITNESS_CTX_MESSAGE", "")
	res, err := run(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ok || res.Errors[0] != msgEmpty {
		t.Fatalf("expected empty-message failure without git fallback, got %+v", res)
	}
}
