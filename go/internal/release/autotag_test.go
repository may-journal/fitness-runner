package release

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func tagFixture(t *testing.T) Config {
	t.Helper()
	c := testConfig(t)
	remote := t.TempDir()
	tagGit(t, remote, "init", "--bare", "-q")
	tagGit(t, c.Root, "init", "-q")
	tagGit(t, c.Root, "config", "user.name", "Fitness")
	tagGit(t, c.Root, "config", "user.email", "fitness@example.com")
	tagGit(t, c.Root, "remote", "add", "origin", remote)
	tagGit(t, c.Root, "add", "CHANGELOG.md")
	tagGit(t, c.Root, "commit", "-qm", "Initial release")
	return c
}

func tagGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = cleanEnvironment()
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}

func TestEnsureTagPublishesAndRepeats(t *testing.T) {
	c := tagFixture(t)
	ctx := context.Background()
	for range 2 {
		if err := c.EnsureTag(ctx); err != nil {
			t.Fatal(err)
		}
	}
	remote, err := c.remoteTag(ctx, "refs/tags/go/v0.20261004.37")
	if err != nil || remote != tagGit(t, c.Root, "rev-parse", "HEAD") {
		t.Fatalf("remote tag = %s, %v", remote, err)
	}
}

func TestEnsureTagRejectsVersionReuse(t *testing.T) {
	c := tagFixture(t)
	ctx := context.Background()
	if err := c.EnsureTag(ctx); err != nil {
		t.Fatal(err)
	}
	before := tagGit(t, c.Root, "rev-parse", "HEAD")
	tagGit(t, c.Root, "commit", "--allow-empty", "-qm", "Changed source")
	if err := c.EnsureTag(ctx); err == nil || !strings.Contains(err.Error(), "add a new changelog version") {
		t.Fatalf("expected version collision: %v", err)
	}
	assertExistingTag(t, c, before)
}

func assertExistingTag(t *testing.T, c Config, before string) {
	t.Helper()
	remote, err := c.remoteTag(context.Background(), "refs/tags/go/v0.20261004.37")
	if err != nil || remote != before {
		t.Fatalf("existing tag changed: %s, %v", remote, err)
	}
}

func TestEnsureTagAcceptsAnnotatedTag(t *testing.T) {
	c := tagFixture(t)
	tagGit(t, c.Root, "tag", "-a", "go/v0.20261004.37", "-m", "Release")
	tagGit(t, c.Root, "push", "origin", "refs/tags/go/v0.20261004.37")
	if err := c.EnsureTag(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureTagDoesNotPushWithInvalidChangelog(t *testing.T) {
	c := tagFixture(t)
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), "# No version\n")
	requireError(t, c.EnsureTag(context.Background()))
	if tags := tagGit(t, c.Root, "ls-remote", "--tags", "origin"); tags != "" {
		t.Fatalf("unexpected published tag: %s", tags)
	}
}

func TestEnsureTagReportsRemoteFailure(t *testing.T) {
	c := tagFixture(t)
	tagGit(t, c.Root, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing"))
	requireError(t, c.EnsureTag(context.Background()))
}

func TestTagRaceAcceptsOnlyExpectedCommit(t *testing.T) {
	c := tagFixture(t)
	ctx := context.Background()
	if err := c.EnsureTag(ctx); err != nil {
		t.Fatal(err)
	}
	head := tagGit(t, c.Root, "rev-parse", "HEAD")
	pushErr := exec.ErrNotFound
	if err := c.checkTagRace(ctx, "refs/tags/go/v0.20261004.37", head, pushErr); err != nil {
		t.Fatal(err)
	}
	if err := c.checkTagRace(ctx, "refs/tags/go/v0.20261004.37", "different", pushErr); err != pushErr {
		t.Fatalf("race error = %v", err)
	}
}

func pinCommitFixture(t *testing.T) Config {
	t.Helper()
	c := tagFixture(t)
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), "# Changelog\n\n## Changes\n\n### 2026.10.04.0037\n\n- Initial\n")
	testWrite(t, filepath.Join(c.Root, "action.yml"), "uses: may-journal/fitness-runner@go/v0.20261004.37\n")
	tagGit(t, c.Root, "add", "CHANGELOG.md", "action.yml")
	tagGit(t, c.Root, "commit", "-qm", "Original pins")
	return c
}

func commitPinUpdate(t *testing.T, c Config, functional bool) {
	t.Helper()
	before := tagGit(t, c.Root, "show", "HEAD:CHANGELOG.md")
	section := "### 2026.10.04.0038\n\n- Chore: update Fitness pins to verified release go/v0.20261004.38.\n- Chore: keep the installer and shared workflows on the same release.\n- Docs: refresh the pinned release used in setup examples.\n\n"
	after := strings.Replace(before, "## Changes\n\n", "## Changes\n\n"+section, 1)
	testWrite(t, filepath.Join(c.Root, "CHANGELOG.md"), after+"\n")
	pin := "uses: may-journal/fitness-runner@go/v0.20261004.38\n"
	if functional {
		pin += "runs: changed\n"
	}
	testWrite(t, filepath.Join(c.Root, "action.yml"), pin)
	tagGit(t, c.Root, "add", "CHANGELOG.md", "action.yml")
	tagGit(t, c.Root, "commit", "-qm", "Update pins\n\nFitness-Pin-Release: go/v0.20261004.38")
}

func TestPinOnlyCommitSkipsRelease(t *testing.T) {
	c := pinCommitFixture(t)
	commitPinUpdate(t, c, false)
	skip, err := c.PinOnlyCommit(context.Background())
	if err != nil || !skip {
		t.Fatalf("pin update not recognized: %v, %v", skip, err)
	}
	if err := c.EnsureTag(context.Background()); err != nil {
		t.Fatal(err)
	}
	if tags := tagGit(t, c.Root, "ls-remote", "--tags", "origin"); tags != "" {
		t.Fatalf("pin update triggered release: %s", tags)
	}
}

func TestPinOnlyCommitRejectsFunctionalChange(t *testing.T) {
	c := pinCommitFixture(t)
	commitPinUpdate(t, c, true)
	skip, err := c.PinOnlyCommit(context.Background())
	if err != nil || skip {
		t.Fatalf("functional change exempted: %v, %v", skip, err)
	}
}

func TestPinOnlyCommitRejectsUnknownFile(t *testing.T) {
	c := pinCommitFixture(t)
	commitPinUpdate(t, c, false)
	testWrite(t, filepath.Join(c.Root, "source.go"), "package changed\n")
	tagGit(t, c.Root, "add", "source.go")
	tagGit(t, c.Root, "commit", "--amend", "--no-edit", "-q")
	skip, err := c.PinOnlyCommit(context.Background())
	if err != nil || skip {
		t.Fatalf("source change exempted: %v, %v", skip, err)
	}
}
