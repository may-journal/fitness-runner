// cspell:ignore autorelease releasemeta
package releasemeta

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var versionHeading = regexp.MustCompile(`(?m)^#{2,} \[?\d+\.\d+\.\d+`)
var bold = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
var issueMarker = regexp.MustCompile(`<!-- fitness-release-issue:(\d+) -->`)

type Client struct{ Root, Repo string }

type pullRequest struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	Body   string `json:"body"`
	Base   struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref  string `json:"ref"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

type issue struct {
	Number      int    `json:"number"`
	Body        string `json:"body"`
	PullRequest any    `json:"pull_request"`
}

func ReleaseNotes(body string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	first, last := lineIndex(lines, "---"), lineLastIndex(lines, "---")
	if first < 0 || last <= first {
		return "", fmt.Errorf("missing Release Please note delimiters")
	}
	notes := strings.TrimSpace(strings.Join(lines[first+1:last], "\n"))
	if !versionHeading.MatchString(notes) {
		return "", fmt.Errorf("missing Release Please version heading")
	}
	notes = regexp.MustCompile(`(?m)^## `).ReplaceAllString(notes, "### ")
	return bold.ReplaceAllString(notes, "$1"), nil
}

func FormatBody(body string, issueNumber int) (string, error) {
	notes, err := ReleaseNotes(body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("> Prepare a reviewed Fitness release.\n\n## Background\n\nRelease Please groups merged changes for the next release. Binary builds and public checks run after this PR merges.\n\nCloses #%d. Agent: Release Please.\n\n<!-- fitness-release-issue:%d -->\n\n## Changelog\n\n---\n\n%s\n\n---\nRelease Please manages this release PR.\n", issueNumber, issueNumber, notes), nil
}

func (c Client) UpdateReleasePR(ctx context.Context, number int) error {
	pr, err := c.loadValidPR(ctx, number)
	if err != nil {
		return err
	}
	tracking, err := c.trackingIssue(ctx, pr)
	if err != nil {
		return err
	}
	body, err := FormatBody(pr.Body, tracking)
	if err != nil {
		return err
	}
	if body == pr.Body {
		return nil
	}
	return c.updatePRBody(ctx, number, body)
}

func (c Client) loadValidPR(ctx context.Context, number int) (pullRequest, error) {
	var pr pullRequest
	if err := c.apiJSON(ctx, &pr, "repos/"+c.Repo+"/pulls/"+strconv.Itoa(number)); err != nil {
		return pr, err
	}
	return pr, validatePR(pr, c.Repo)
}

func (c Client) updatePRBody(ctx context.Context, number int, body string) error {
	data, _ := json.Marshal(map[string]string{"body": body})
	_, err := c.api(ctx, "--method", "PATCH", "repos/"+c.Repo+"/pulls/"+strconv.Itoa(number), "--input", "-", string(data))
	return err
}

func (c Client) EnsureTrackingIssue(ctx context.Context, key, title, body string) (int, error) {
	marker := "<!-- fitness-release-tracking:" + key + " -->"
	var issues []issue
	if err := c.apiJSON(ctx, &issues, "repos/"+c.Repo+"/issues?state=all&per_page=100"); err != nil {
		return 0, err
	}
	if number := matchingIssue(issues, marker); number != 0 {
		return number, nil
	}
	return c.createIssue(ctx, title, body+"\n\n"+marker)
}

func matchingIssue(issues []issue, marker string) int {
	for _, candidate := range issues {
		if candidate.PullRequest == nil && strings.Contains(candidate.Body, marker) {
			return candidate.Number
		}
	}
	return 0
}

func (c Client) createIssue(ctx context.Context, title, body string) (int, error) {
	data, _ := json.Marshal(map[string]string{"title": title, "body": body})
	var created issue
	if err := c.apiJSONInput(ctx, &created, "repos/"+c.Repo+"/issues", data); err != nil {
		return 0, err
	}
	return created.Number, nil
}

func (c Client) trackingIssue(ctx context.Context, pr pullRequest) (int, error) {
	if match := issueMarker.FindStringSubmatch(pr.Body); match != nil {
		return strconv.Atoi(match[1])
	}
	return c.EnsureTrackingIssue(ctx, "pr-"+strconv.Itoa(pr.Number), "Publish Fitness release from PR #"+strconv.Itoa(pr.Number), "Track the reviewed release in #"+strconv.Itoa(pr.Number)+". Release Please owns the version and notes; GoReleaser builds the binaries.")
}

func validatePR(pr pullRequest, repo string) error {
	if pr.State != "open" || pr.Base.Ref != "main" {
		return fmt.Errorf("release PR must target main and be open")
	}
	if !validReleaseHead(pr, repo) {
		return fmt.Errorf("unexpected release PR source")
	}
	if !hasPendingLabel(pr) {
		return fmt.Errorf("missing release lifecycle label")
	}
	_, err := ReleaseNotes(pr.Body)
	return err
}

func validReleaseHead(pr pullRequest, repo string) bool {
	return pr.Head.Repo.FullName == repo && strings.HasPrefix(pr.Head.Ref, "release-please--branches--main")
}
func hasPendingLabel(pr pullRequest) bool {
	for _, label := range pr.Labels {
		if label.Name == "autorelease: pending" {
			return true
		}
	}
	return false
}

func (c Client) apiJSON(ctx context.Context, target any, endpoint string) error {
	out, err := c.api(ctx, endpoint)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, target)
}

func (c Client) apiJSONInput(ctx context.Context, target any, endpoint string, data []byte) error {
	out, err := c.api(ctx, "--method", "POST", endpoint, "--input", "-", string(data))
	if err != nil {
		return err
	}
	return json.Unmarshal(out, target)
}

func (c Client) api(ctx context.Context, args ...string) ([]byte, error) {
	input := ""
	if len(args) > 1 && args[len(args)-2] == "-" {
		input, args = args[len(args)-1], args[:len(args)-1]
	}
	cmd := exec.CommandContext(ctx, "gh", append([]string{"api"}, args...)...)
	cmd.Dir = c.Root
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func lineIndex(lines []string, target string) int {
	for i, line := range lines {
		if line == target {
			return i
		}
	}
	return -1
}
func lineLastIndex(lines []string, target string) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] == target {
			return i
		}
	}
	return -1
}
