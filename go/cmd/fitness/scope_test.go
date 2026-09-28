package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// gitRepo makes a throwaway repo with one committed file and a.md staged.
func gitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "add", "a.md").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	return root
}

func TestChangedScope(t *testing.T) {
	root := gitRepo(t)
	cases := []struct {
		name, baseRef, actions string
		all                    bool
		want                   []string
	}{
		{name: "local commit scopes to staged files", want: []string{"a.md"}},
		{name: "--all scans everything", all: true},
		{name: "push in CI scans everything", actions: "true"},
		{name: "unresolvable PR base scans everything", baseRef: "main", actions: "true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GITHUB_BASE_REF", tc.baseRef)
			t.Setenv("GITHUB_ACTIONS", tc.actions)
			if got := changedScope(root, tc.all); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("changedScope = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestChangedScopePullRequestDiff(t *testing.T) {
	root := gitRepo(t)
	for _, args := range [][]string{
		{"update-ref", "refs/remotes/origin/main", "HEAD"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "add a"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	t.Setenv("GITHUB_BASE_REF", "main")
	t.Setenv("GITHUB_ACTIONS", "true")
	if got := changedScope(root, false); !reflect.DeepEqual(got, []string{"a.md"}) {
		t.Fatalf("changedScope = %v, want [a.md]", got)
	}
}

func TestContextEnvDropsInheritedScope(t *testing.T) {
	t.Setenv("FITNESS_CHANGED_FILES", "stale.md")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_ACTIONS", "")
	for _, kv := range contextEnv(t.TempDir(), nil, true) {
		if kv == "FITNESS_CHANGED_FILES=stale.md" {
			t.Fatal("inherited FITNESS_CHANGED_FILES leaked into a full scan")
		}
	}
}

func TestAllFlagIsARunnerArg(t *testing.T) {
	if !hasAllFlag([]string{"--jobs=2", "--all"}) {
		t.Fatal("--all not detected")
	}
	_, _, _, passthrough := parseArgv([]string{"cspell", "--all", "x"})
	if !reflect.DeepEqual(passthrough, []string{"x"}) {
		t.Fatalf("passthrough = %v, want [x]", passthrough)
	}
}
