package main

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestPlanNumbers(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"single", "feat: x\n\nPlan #12\n", []string{"12"}},
		{"case and spacing", "chore: y\n\nplan #  7", []string{"7"}},
		{"distinct in first-seen order", "Plan #3\nPlan #9\nPlan #3", []string{"3", "9"}},
		{"none", "just a body, no plan reference", nil},
		{"not a bare number", "we plan 5 steps", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := planNumbers(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("planNumbers = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAllChoreDocsSubjects(t *testing.T) {
	cases := []struct {
		name     string
		subjects []string
		want     bool
	}{
		{"all chore/docs", []string{"chore(ci): x", "docs: y"}, true},
		{"one feat", []string{"chore: x", "feat: y"}, false},
		{"empty is vacuously true", nil, true},
		{"fix is not exempt", []string{"fix: bug"}, false},
		{"merge is exempt", []string{"Merge branch 'main' into topic", "chore: x"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allChoreDocsSubjects(tc.subjects); got != tc.want {
				t.Fatalf("allChoreDocsSubjects(%v) = %v, want %v", tc.subjects, got, tc.want)
			}
		})
	}
}

func TestCommitsOfSkipsCommitsAlreadyOnOrigin(t *testing.T) {
	remote := t.TempDir()
	dir := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "--bare", remote)
	git("init", "-q", "-b", "main")
	git("remote", "add", "origin", remote)
	git("commit", "-q", "--allow-empty", "-m", "chore: root")
	git("push", "-q", "origin", "main")

	// A branch is pushed; then main gains a feat commit on origin.
	git("switch", "-q", "-c", "topic")
	git("commit", "-q", "--allow-empty", "-m", "chore: topic work")
	git("push", "-q", "origin", "topic")
	remoteTip := git("rev-parse", "HEAD")
	git("switch", "-q", "main")
	git("commit", "-q", "--allow-empty", "-m", "feat: landed on main")
	git("push", "-q", "origin", "main")

	// Merging main into the branch pushes only the merge and new chore work.
	git("switch", "-q", "topic")
	git("merge", "-q", "--no-edit", "main")
	git("commit", "-q", "--allow-empty", "-m", "chore: follow-up")
	t.Chdir(dir)

	commits := commitsOf(git("rev-parse", "HEAD"), remoteTip)
	if len(commits) != 2 {
		t.Fatalf("commitsOf = %d commits, want 2 (merge + follow-up)", len(commits))
	}
	if !allChoreDocs(commits) {
		t.Fatal("the feat commit already on origin/main leaked into the gate")
	}

	// A new branch still counts only what origin lacks.
	if got := commitsOf(git("rev-parse", "HEAD"), zeroSha); len(got) != 2 {
		t.Fatalf("new-branch commitsOf = %d commits, want 2", len(got))
	}
}
