package release

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pinPRVersion = "0.20261004.1200"

func pinPRFixture(t *testing.T) (Config, string) {
	t.Helper()
	c := tagFixture(t)
	pins := pinFixture(t)
	for _, path := range PinFiles() {
		data, err := os.ReadFile(filepath.Join(pins.Root, path))
		if err != nil {
			t.Fatal(err)
		}
		testWrite(t, filepath.Join(c.Root, path), string(data))
	}
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), "# Changelog\n\n## Changes\n\n### 2026.10.04.1140\n\n- Initial release\n")
	tagGit(t, c.Root, "add", ".")
	tagGit(t, c.Root, "commit", "-qm", "Add release pins")
	tagGit(t, c.Root, "branch", "-M", "main")
	tagGit(t, c.Root, "push", "-u", "origin", "main")
	if err := c.saveMetadata(Metadata{Version: pinPRVersion}); err != nil {
		t.Fatal(err)
	}
	return c, fakePinGH(t)
}

func fakePinGH(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "gh")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$FITNESS_PIN_TEST_STATE/calls"
case "$1 $2" in
  'release view')
    if test -f "$FITNESS_PIN_TEST_STATE/latest"; then cat "$FITNESS_PIN_TEST_STATE/latest"; else printf '{"tagName":"go/v0.20261004.1200"}'; fi ;;
  'pr list')
    case " $* " in
      *' --head '*)
        if test -f "$FITNESS_PIN_TEST_STATE/existing"; then cat "$FITNESS_PIN_TEST_STATE/existing"; else printf '[]'; fi ;;
      *)
        if test -f "$FITNESS_PIN_TEST_STATE/older"; then cat "$FITNESS_PIN_TEST_STATE/older"; else printf '[]'; fi ;;
    esac ;;
  'issue list') printf '[{"number":42,"title":"Update Fitness pins to go/v0.20261004.1200"}]' ;;
  'pr create')
    if test -f "$FITNESS_PIN_TEST_STATE/fail-create"; then exit 1; fi
    printf '[{"number":43,"url":"https://example.com/pull/43"}]' > "$FITNESS_PIN_TEST_STATE/existing"
    printf 'https://example.com/pull/43' ;;
  'pr close')
    if test -f "$FITNESS_PIN_TEST_STATE/fail-close"; then exit 1; fi
    exit 0 ;;
  *) exit 9 ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FITNESS_PIN_TEST_STATE", directory)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	return directory
}

func pinCalls(t *testing.T, state string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(state, "calls"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPinReleaseCreatesLoopSafeCommit(t *testing.T) {
	c, state := pinPRFixture(t)
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	valid, err := c.PinOnlyCommit(context.Background())
	if err != nil || !valid {
		t.Fatalf("generated branch triggers another release: %v, %v", valid, err)
	}
	if !strings.Contains(pinCalls(t, state), "pr create --base main --head "+pinBranchPrefix+pinPRVersion) {
		t.Fatal("pin PR was not opened against main")
	}
}

func TestPinReleaseRetryPreservesReviewEdits(t *testing.T) {
	c, state := pinPRFixture(t)
	ctx := context.Background()
	if err := c.PinRelease(ctx); err != nil {
		t.Fatal(err)
	}
	testWrite(t, filepath.Join(c.Root, "review.txt"), "A reviewer edit\n")
	tagGit(t, c.Root, "add", "review.txt")
	tagGit(t, c.Root, "commit", "-qm", "Review edit")
	tagGit(t, c.Root, "push", "origin", "HEAD:refs/heads/"+pinBranchPrefix+pinPRVersion)
	head := tagGit(t, c.Root, "rev-parse", "HEAD")
	if err := c.PinRelease(ctx); err != nil {
		t.Fatal(err)
	}
	if tagGit(t, c.Root, "rev-parse", "HEAD") != head || strings.Count(pinCalls(t, state), "pr create ") != 1 {
		t.Fatal("retry replaced review edits or duplicated the pull request")
	}
}

func TestPinReleaseRecoversAfterPRCreationFails(t *testing.T) {
	c, state := pinPRFixture(t)
	testWrite(t, filepath.Join(state, "fail-create"), "fail")
	requireError(t, c.PinRelease(context.Background()))
	head := tagGit(t, c.Root, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(state, "fail-create")); err != nil {
		t.Fatal(err)
	}
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if tagGit(t, c.Root, "rev-parse", "HEAD") != head || strings.Count(pinCalls(t, state), "pr create ") != 2 {
		t.Fatal("retry did not reuse the pushed commit")
	}
}

func TestPinReleaseRejectsUnfamiliarBranch(t *testing.T) {
	c, state := pinPRFixture(t)
	branch := "refs/heads/" + pinBranchPrefix + pinPRVersion
	head := tagGit(t, c.Root, "rev-parse", "HEAD")
	tagGit(t, c.Root, "push", "origin", "HEAD:"+branch)
	requireError(t, c.PinRelease(context.Background()))
	remote := tagGit(t, c.Root, "ls-remote", "origin", branch)
	if !strings.HasPrefix(remote, head) || strings.Contains(pinCalls(t, state), "pr create ") {
		t.Fatal("unfamiliar branch was overwritten or submitted for review")
	}
}

func TestPinReleaseSkipsOlderVersion(t *testing.T) {
	c, state := pinPRFixture(t)
	if err := c.saveMetadata(Metadata{Version: "0.20261004.1000"}); err != nil {
		t.Fatal(err)
	}
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pinCalls(t, state), "pr create ") {
		t.Fatal("stale release created a downgrade PR")
	}
}

func TestPinReleaseClosesOnlyItsOlderPRs(t *testing.T) {
	c, state := pinPRFixture(t)
	olderHead := createOlderPinBranch(t, c)
	prs := `[
{"number":1,"headRefOid":"` + olderHead + `","headRefName":"codex/release-pins-0.20261004.1150","body":"<!-- fitness:release-pins:0.20261004.1150 -->"},
{"number":2,"headRefName":"codex/release-pins-0.20261004.1140","body":"Manual update"},
{"number":3,"headRefName":"codex/release-pins-0.20261004.1140","body":"<!-- fitness:release-pins:0.20261004.1140 -->","isCrossRepository":true},
{"number":4,"headRefName":"codex/release-pins-0.20261004.1300","body":"<!-- fitness:release-pins:0.20261004.1300 -->"},
{"number":5,"headRefName":"other-branch","body":"<!-- fitness:release-pins:0.20261004.1140 -->"}
]`
	testWrite(t, filepath.Join(state, "older"), prs)
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls := pinCalls(t, state)
	if strings.Count(calls, "pr close ") != 1 || !strings.Contains(calls, "pr close 1 ") {
		t.Fatalf("unrelated PRs affected:\n%s", calls)
	}
}

func TestPinReleaseRetrySkipsSupersededBranch(t *testing.T) {
	c, state := pinPRFixture(t)
	testWrite(t, filepath.Join(state, "fail-create"), "fail")
	requireError(t, c.PinRelease(context.Background()))
	advanceMainPins(t, c, "0.20261004.1300")
	if err := os.Remove(filepath.Join(state, "fail-create")); err != nil {
		t.Fatal(err)
	}
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Count(pinCalls(t, state), "pr create ") != 1 {
		t.Fatal("retry opened a superseded pin branch")
	}
}

func advanceMainPins(t *testing.T, c Config, version string) {
	t.Helper()
	tagGit(t, c.Root, "checkout", "main")
	if _, err := c.UpdatePins(version); err != nil {
		t.Fatal(err)
	}
	tagGit(t, c.Root, "add", "--", "action.yml", "go/cmd/fitness-install/main.go", ".github", "docs", "README.md")
	tagGit(t, c.Root, "commit", "-qm", "Advance pins")
	tagGit(t, c.Root, "push", "origin", "main")
}

func createOlderPinBranch(t *testing.T, c Config) string {
	t.Helper()
	version := "0.20261004.1150"
	ready, err := c.createPinBranch(context.Background(), version, pinBranchPrefix+version)
	if err != nil || !ready {
		t.Fatalf("older branch: %v, %v", ready, err)
	}
	head := tagGit(t, c.Root, "rev-parse", "HEAD")
	tagGit(t, c.Root, "checkout", "main")
	return head
}

func TestPinReleaseSkipsOlderThanPublished(t *testing.T) {
	c, state := pinPRFixture(t)
	testWrite(t, filepath.Join(state, "latest"), `{"tagName":"go/v0.20261004.1300"}`)
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pinCalls(t, state), "pr create ") {
		t.Fatal("superseded release opened a PR")
	}
}

func TestPinReleasePreservesOlderReviewEdits(t *testing.T) {
	c, state := pinPRFixture(t)
	createOlderPinBranch(t, c)
	branch := pinBranchPrefix + "0.20261004.1150"
	tagGit(t, c.Root, "checkout", branch)
	testWrite(t, filepath.Join(c.Root, "review.txt"), "Keep this review change\n")
	tagGit(t, c.Root, "add", "review.txt")
	tagGit(t, c.Root, "commit", "-qm", "Review changes")
	tagGit(t, c.Root, "push", "origin", "HEAD:refs/heads/"+branch)
	head := tagGit(t, c.Root, "rev-parse", "HEAD")
	tagGit(t, c.Root, "checkout", "main")
	setOlderPinPR(t, state, head)
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(pinCalls(t, state), "pr close ") {
		t.Fatal("closed PR with review changes")
	}
}

func setOlderPinPR(t *testing.T, state, head string) {
	t.Helper()
	testWrite(t, filepath.Join(state, "older"), `[{"number":1,"headRefOid":"`+head+`","headRefName":"codex/release-pins-0.20261004.1150","body":"<!-- fitness:release-pins:0.20261004.1150 -->"}]`)
}

func TestPinReleaseRetryCompletesCleanup(t *testing.T) {
	c, state := pinPRFixture(t)
	setOlderPinPR(t, state, createOlderPinBranch(t, c))
	testWrite(t, filepath.Join(state, "fail-close"), "fail")
	requireError(t, c.PinRelease(context.Background()))
	if err := os.Remove(filepath.Join(state, "fail-close")); err != nil {
		t.Fatal(err)
	}
	if err := c.PinRelease(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls := pinCalls(t, state)
	if strings.Count(calls, "pr create ") != 1 || strings.Count(calls, "pr close 1 ") != 2 {
		t.Fatalf("retry calls: %s", calls)
	}
}
