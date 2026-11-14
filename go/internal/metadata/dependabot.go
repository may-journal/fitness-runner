package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/pkg/changelog"
)

// dependabotLogin is the account whose pull requests dependabot-pr completes.
const dependabotLogin = "dependabot[bot]"

// dependabotEvent is the pull_request_target payload dependabot-pr reads.
type dependabotEvent struct {
	PullRequest struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
	} `json:"pull_request"`
}

// bumpRe finds the dependency or group a Dependabot title bumps.
var bumpRe = regexp.MustCompile(`(?i)\bbump (?:the )?(.+?)(?: from | in | with | group|$)`)

// RunDependabotPR completes a Dependabot pull request, so every PR check judges it as it judges a person's.
func RunDependabotPR() int {
	data, err := os.ReadFile(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		return ghFailure("dependabot-pr", fmt.Errorf("read the event payload: %w", err))
	}
	var ev dependabotEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return ghFailure("dependabot-pr", err)
	}
	gh, err := aftereffect.NewClient()
	if err != nil {
		return ghFailure("dependabot-pr", err)
	}
	root, _ := os.Getwd()
	if err := completeBotPR(gh, blobChecker{root}, ev, time.Now().UTC()); err != nil {
		return ghFailure("dependabot-pr", err)
	}
	return 0
}

// completeBotPR fixes the title, finds the bump issue, commits the changelog entry and rewrites the description.
func completeBotPR(gh aftereffect.Client, b blobChecker, ev dependabotEvent, now time.Time) error {
	pr := ev.PullRequest
	if !strings.EqualFold(pr.User.Login, dependabotLogin) {
		fmt.Printf("fitness dependabot-pr: #%d is not a Dependabot pull request, so it is left alone.\n", pr.Number)
		return nil
	}
	title, err := fixedTitle(pr.Title, b)
	if err != nil {
		return err
	}
	file, err := commitEntry(gh, pr.Number, pr.Head.Ref, entryPath(now, bumpName(title)), now, title)
	if err != nil {
		return err
	}
	body, err := completedBody(gh, pr.Body, file, title, pr.Head.Ref, pr.Number)
	if err != nil {
		return err
	}
	return editIfChanged(gh, pr.Number, pr.Title, pr.Body, title, body)
}

// fixedTitle is the PR's title made semantic, or an error when it still fails.
func fixedTitle(title string, b blobChecker) (string, error) {
	title = semanticTitle(title, b)
	if errs := b.errorsOf("semantic-commit", "--message", title); len(errs) > 0 {
		return "", fmt.Errorf("the title %q still fails: %s", title, strings.Join(errs, "; "))
	}
	return title, nil
}

// completedBody is body when it closes an issue and links file, else
// the template filled in around Dependabot's notes with the bump issue.
func completedBody(gh aftereffect.Client, body, file, title, branch string, pr int) (string, error) {
	if strings.Contains(body, file) && strings.Contains(body, "Closes #") {
		return body, nil
	}
	issue, err := bumpIssue(gh, bumpName(title), pr)
	return botBody(gh.Repo(), branch, file, title, issue, body), err
}

// editIfChanged edits the PR only when its title or body differ.
func editIfChanged(gh aftereffect.Client, n int, oldTitle, oldBody, title, body string) error {
	if oldTitle == title && oldBody == body {
		return nil
	}
	return gh.EditPR(n, title, body)
}

// semanticTitle is title when the semantic-commit check accepts it, else
// title as a `build(deps)` commit.
func semanticTitle(title string, b blobChecker) string {
	if len(b.errorsOf("semantic-commit", "--message", title)) == 0 {
		return title
	}
	rs := []rune(strings.TrimSpace(title))
	if len(rs) > 0 {
		rs[0] = unicode.ToLower(rs[0])
	}
	return "build(deps): " + string(rs)
}

// bumpName is the dependency or group a title bumps, or the whole title.
func bumpName(title string) string {
	if m := bumpRe.FindStringSubmatch(title); m != nil {
		return m[1]
	}
	return title
}

// bumpIssue is the open issue for bumping name, opened when none exists.
func bumpIssue(gh aftereffect.Client, name string, pr int) (int, error) {
	title := "Bump " + name
	n, err := gh.OpenIssueNumber(title)
	if err != nil || n != 0 {
		return n, err
	}
	body := fmt.Sprintf("Dependabot proposed bumping %s in https://github.com/%s/pull/%d\n", name, gh.Repo(), pr)
	return gh.CreateIssue(title, body)
}

// entryPath is the changelog entry file for a bump made at now.
func entryPath(now time.Time, name string) string {
	slug := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	return fmt.Sprintf("%s/%s-bump-%s.md", changelog.EntryDir, now.Format("2006-01-02-1504"), slug)
}

// commitEntry adds the changelog entry to the PR branch and returns its
// path, or returns the entry the PR already has.
func commitEntry(gh aftereffect.Client, pr int, branch, file string, now time.Time, title string) (string, error) {
	have, err := gh.DirFiles(pr, changelog.EntryDir)
	if err != nil {
		return "", err
	}
	for _, f := range have {
		if changelog.IsEntryPath(f.Filename) {
			return f.Filename, nil
		}
	}
	text := strings.TrimPrefix(title, "build(deps): ")
	entry := fmt.Sprintf("### %s\n\n> A dependency update keeps the project current.\n\n- Build: %s.\n- Chore: Dependabot proposed the change.\n- Chore: the pull request closes its tracking issue.\n", now.Format("2006.01.02.1504"), text)
	return file, gh.PutFile(branch, file, "docs(changelog): add the entry for a dependency bump", entry)
}

// botBody is the PR template filled in: a pitch, Dependabot's notes under
// Background in a fenced block, and the changelog link.
func botBody(repo, branch, file, title string, issue int, notes string) string {
	pitch := strings.TrimPrefix(title, "build(deps): ")
	var b strings.Builder
	fence := notesFence(notes)
	fmt.Fprintf(&b, "> %s.\n\n## Background\n\nCloses #%d.\n\nDependabot wrote these release notes:\n\n%s%s\n%s\n%s\n\n", pitch, issue, fence, "text", strings.TrimSpace(notes), fence)
	fmt.Fprintf(&b, "## Changelog\n\n- [Changelog entry](https://github.com/%s/blob/%s/%s)\n", repo, branch, file)
	return b.String()
}

// notesFence is a code fence longer than any run of backticks in notes.
func notesFence(notes string) string {
	longest := 0
	for _, run := range regexp.MustCompile("`+").FindAllString(notes, -1) {
		longest = max(longest, len(run))
	}
	return strings.Repeat("`", max(3, longest+1))
}
