package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const pinBranchPrefix = "codex/release-pins-"

type pinPR struct {
	Number            int    `json:"number"`
	State             string `json:"state"`
	Body              string `json:"body"`
	URL               string `json:"url"`
	HeadRefOid        string `json:"headRefOid"`
	HeadRefName       string `json:"headRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
}

// PinRelease opens a reviewed pin update after the release's public checks pass.
// Each release owns one branch; retries reuse it without replacing review edits.
func (c Config) PinRelease(ctx context.Context) error {
	metadata, err := c.Metadata()
	if err != nil {
		return err
	}
	if !pinVersionPattern.MatchString(metadata.Version) {
		return fmt.Errorf("invalid metadata version %q", metadata.Version)
	}
	latest, err := c.latestPublishedVersion(ctx)
	if err != nil {
		return err
	}
	if comparePinVersions(latest, metadata.Version) > 0 {
		fmt.Println("A newer verified release exists; skipping this pin update.")
		return nil
	}
	return c.pinReleaseVersion(ctx, metadata.Version)
}

func (c Config) pinReleaseVersion(ctx context.Context, version string) error {
	branch := pinBranchPrefix + version
	existing, err := c.pinPRs(ctx, "--state", "all", "--head", branch)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		fmt.Println("Pin update already exists: " + existing[0].URL)
		return c.closeOlderPinPRs(ctx, version)
	}
	ready, err := c.preparePinBranch(ctx, version, branch)
	if err != nil || !ready {
		return err
	}
	return c.openPinPR(ctx, version, branch)
}

func (c Config) preparePinBranch(ctx context.Context, version, branch string) (bool, error) {
	if err := c.requireCleanCheckout(ctx); err != nil {
		return false, err
	}
	if _, err := c.releaseGit(ctx, "fetch", "origin", "main"); err != nil {
		return false, err
	}
	needed, err := c.mainNeedsPins(ctx, version)
	if err != nil || !needed {
		return false, err
	}
	return c.selectPinBranch(ctx, version, branch)
}

func (c Config) selectPinBranch(ctx context.Context, version, branch string) (bool, error) {
	remote, err := c.releaseGit(ctx, "ls-remote", "--heads", "origin", "refs/heads/"+branch)
	if err != nil {
		return false, err
	}
	if remote != "" {
		return c.reusePinBranch(ctx, version, branch)
	}
	return c.createPinBranch(ctx, version, branch)
}

func (c Config) requireCleanCheckout(ctx context.Context) error {
	changes, err := c.releaseGit(ctx, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return err
	}
	if changes != "" {
		return fmt.Errorf("pin updates require a clean checkout")
	}
	return nil
}

func (c Config) reusePinBranch(ctx context.Context, version, branch string) (bool, error) {
	if _, err := c.releaseGit(ctx, "fetch", "origin", branch); err != nil {
		return false, err
	}
	if _, err := c.releaseGit(ctx, "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return false, err
	}
	valid, err := c.PinOnlyCommit(ctx)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, fmt.Errorf("existing pin branch %s has unexpected changes; leaving it untouched", branch)
	}
	return c.checkReusedVersion(ctx, version)
}

func (c Config) checkReusedVersion(ctx context.Context, version string) (bool, error) {
	message, err := c.releaseGit(ctx, "show", "-s", "--format=%B", "HEAD")
	if err != nil {
		return false, err
	}
	if match := pinTrailer.FindStringSubmatch(message); len(match) != 2 || match[1] != version {
		return false, fmt.Errorf("existing pin branch has a different release marker")
	}
	_, err = c.releaseGit(ctx, "merge-base", "--is-ancestor", "HEAD^", "origin/main")
	return err == nil, err
}

func (c Config) createPinBranch(ctx context.Context, version, branch string) (bool, error) {
	if _, err := c.releaseGit(ctx, "checkout", "-b", branch, "origin/main"); err != nil {
		return false, err
	}
	changed, err := c.UpdatePins(version)
	if errors.Is(err, ErrPinDowngrade) {
		fmt.Println("A newer release is already pinned; skipping this update.")
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(changed) == 0 {
		fmt.Println("Release pins are already current.")
		return false, nil
	}
	return c.commitPinBranch(ctx, version, branch, changed)
}

func pinNotes(version string) string {
	return "- Chore: update Fitness pins to verified release go/v" + version + ".\n- Chore: keep the installer and shared workflows on the same release.\n- Docs: refresh the pinned release used in setup examples.\n"
}

func (c Config) addPinChangelog(version string) error {
	path := filepath.Join(c.Root, "CHANGELOG.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	const marker = "\n## Changes\n\n"
	if !strings.Contains(string(data), marker) {
		return fmt.Errorf("CHANGELOG.md has no Changes section")
	}
	section := "### " + time.Now().UTC().Format("2006.01.02.1504") + "\n\n" + pinNotes(version) + "\n"
	updated := strings.Replace(string(data), marker, marker+section, 1)
	return os.WriteFile(path, []byte(updated), 0644)
}

func (c Config) commitPinBranch(ctx context.Context, version, branch string, changed []string) (bool, error) {
	if err := c.addPinChangelog(version); err != nil {
		return false, err
	}
	files := append(changed, "CHANGELOG.md")
	if _, err := c.releaseGit(ctx, append([]string{"add", "--"}, files...)...); err != nil {
		return false, err
	}
	message := "chore(release): update Fitness pins to " + version + "\n\nFitness-Pin-Release: go/v" + version
	args := []string{"-c", "user.name=may-journal-automation[bot]", "-c", "user.email=may-journal-automation[bot]@users.noreply.github.com", "commit", "-m", message}
	if _, err := c.releaseGit(ctx, args...); err != nil {
		return false, err
	}
	_, err := c.releaseGit(ctx, "push", "origin", "HEAD:refs/heads/"+branch)
	return err == nil, err
}

func pinMarker(version string) string { return "<!-- fitness:release-pins:" + version + " -->" }

func (c Config) openPinPR(ctx context.Context, version, branch string) error {
	issue, err := c.pinIssue(ctx, version)
	if err != nil {
		return err
	}
	body := "> Keep Fitness users on the verified release.\n\n## Background\n\nThe public release passed all four platform checks. Update the installer, shared workflows, and setup examples to use it.\n\nFixes #" + fmt.Sprint(issue) + ". Agent: Fitness release automation.\n\nFitness-Pin-Release: go/v" + version + "\n\n" + pinMarker(version) + "\n\n## Changelog\n\n" + pinNotes(version)
	output, err := c.ghBody(ctx, body, "pr", "create", "--base", "main", "--head", branch, "--title", "chore(release): update Fitness pins to "+version)
	if err != nil {
		return err
	}
	fmt.Println(output)
	return c.closeOlderPinPRs(ctx, version)
}

func (c Config) pinIssue(ctx context.Context, version string) (int, error) {
	title := "Update Fitness pins to go/v" + version
	output, err := c.releaseGH(ctx, "issue", "list", "--state", "all", "--search", title+" in:title", "--json", "number,title")
	if err != nil {
		return 0, err
	}
	var issues []struct {
		Number int
		Title  string
	}
	if err = json.Unmarshal([]byte(output), &issues); err != nil {
		return 0, err
	}
	for _, issue := range issues {
		if issue.Title != title {
			continue
		}
		return issue.Number, nil
	}
	return c.createPinIssue(ctx, title, version)
}

func (c Config) createPinIssue(ctx context.Context, title, version string) (int, error) {
	body := "Update the root action, installer, shared workflows, and setup docs to verified release go/v" + version + ". The release passed all four public platform checks. This issue tracks the generated pin PR."
	link, err := c.ghBody(ctx, body, "issue", "create", "--title", title)
	if err != nil {
		return 0, err
	}
	output, err := c.releaseGH(ctx, "issue", "view", strings.TrimSpace(link), "--json", "number")
	if err != nil {
		return 0, err
	}
	var issue struct{ Number int }
	err = json.Unmarshal([]byte(output), &issue)
	return issue.Number, err
}

func (c Config) pinPRs(ctx context.Context, args ...string) ([]pinPR, error) {
	command := append([]string{"pr", "list"}, args...)
	command = append(command, "--limit", "100", "--json", "number,state,body,url,headRefName,headRefOid,isCrossRepository")
	output, err := c.releaseGH(ctx, command...)
	if err != nil {
		return nil, err
	}
	var prs []pinPR
	err = json.Unmarshal([]byte(output), &prs)
	return prs, err
}

func (c Config) closeOlderPinPRs(ctx context.Context, version string) error {
	prs, err := c.pinPRs(ctx, "--state", "open")
	if err != nil {
		return err
	}
	for _, pr := range prs {
		if !olderPinPR(pr, version) {
			continue
		}
		if err := c.closeUntouchedPinPR(ctx, pr, version); err != nil {
			return err
		}
	}
	return nil
}

func olderPinPR(pr pinPR, version string) bool {
	if !strings.HasPrefix(pr.HeadRefName, pinBranchPrefix) {
		return false
	}
	old := strings.TrimPrefix(pr.HeadRefName, pinBranchPrefix)
	if pr.IsCrossRepository || !pinVersionPattern.MatchString(old) {
		return false
	}
	return strings.Contains(pr.Body, pinMarker(old)) && comparePinVersions(old, version) < 0
}

func (c Config) ghBody(ctx context.Context, body string, args ...string) (string, error) {
	file, err := os.CreateTemp("", "fitness-release-body-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	if _, err = file.WriteString(body); err != nil {
		file.Close()
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	return c.releaseGH(ctx, append(args, "--body-file", file.Name())...)
}

func (c Config) releaseGH(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = c.Root
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(output)), nil
}

func (c Config) mainNeedsPins(ctx context.Context, version string) (bool, error) {
	needed := false
	for _, path := range PinFiles() {
		text, err := c.releaseGit(ctx, "show", "origin/main:"+path)
		if err != nil {
			return false, err
		}
		if err := checkPinVersions(path, text, version); err != nil {
			return pinVersionNeeded(err)
		}
		needed = needed || replacePins(path, text, version) != text
	}
	return needed, nil
}

func pinVersionNeeded(err error) (bool, error) {
	if errors.Is(err, ErrPinDowngrade) {
		fmt.Println("A newer release is already pinned; skipping this update.")
		return false, nil
	}
	return false, err
}

func (c Config) closeUntouchedPinPR(ctx context.Context, pr pinPR, version string) error {
	valid, err := c.untouchedPinPR(ctx, pr)
	if err != nil {
		return err
	}
	if !valid {
		return nil
	}
	_, err = c.releaseGH(ctx, "pr", "close", fmt.Sprint(pr.Number), "--comment", "Superseded by verified Fitness release go/v"+version+".")
	return err
}

func (c Config) untouchedPinPR(ctx context.Context, pr pinPR) (bool, error) {
	if pr.HeadRefOid == "" {
		return false, nil
	}
	if _, err := c.releaseGit(ctx, "fetch", "origin", "refs/heads/"+pr.HeadRefName); err != nil {
		return false, err
	}
	remote, err := c.releaseGit(ctx, "rev-parse", "FETCH_HEAD")
	if err != nil {
		return false, err
	}
	if remote != pr.HeadRefOid {
		return false, nil
	}
	return c.generatedPinRevision(ctx, remote)
}

func (c Config) generatedPinRevision(ctx context.Context, revision string) (bool, error) {
	count, err := c.releaseGit(ctx, "rev-list", "--count", "origin/main.."+revision)
	if err != nil {
		return false, err
	}
	if count != "1" {
		return false, nil
	}
	return c.pinOnlyRevision(ctx, revision)
}
