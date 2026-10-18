package metadata

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/aftereffect"
	"github.com/may-journal/fitness-runner/go/internal/gitx"
)

// branchContexts uses push destinations and local tips only inside pre-push.
func branchContexts(ctx context.Context, root string) ([]gitx.BranchContext, error) {
	if os.Getenv("FITNESS_INTERNAL_HOOK") == "pre-push" {
		var branches []gitx.BranchContext
		err := json.Unmarshal([]byte(os.Getenv("FITNESS_PUSH_BRANCHES")), &branches)
		return branches, err
	}
	name, err := currentBranch(ctx, root)
	if err != nil || name == "" {
		return nil, err
	}
	return []gitx.BranchContext{{Name: name, Tip: "HEAD"}}, nil
}

// PushIssuesEnv names the variable through which the pre-push gate hands
// local-branch the issues it already read, as a JSON list of PushIssue.
const PushIssuesEnv = "FITNESS_PUSH_ISSUES"

// PushIssue is one issue the pre-push gate read: its repository, number,
// body and label names.
type PushIssue struct {
	Repo   string   `json:"repo"`
	Number int      `json:"number"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

// PushIssueAt builds a PushIssue from an issue's GitHub URL, as in
// https://github.com/owner/repo/issues/12; ok is false for any other URL.
func PushIssueAt(issueURL, body string, labels []string) (PushIssue, bool) {
	path, ok := strings.CutPrefix(issueURL, "https://github.com/")
	parts := strings.Split(path, "/")
	if !ok || len(parts) != 4 || parts[2] != "issues" {
		return PushIssue{}, false
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil {
		return PushIssue{}, false
	}
	return PushIssue{Repo: parts[0] + "/" + parts[1], Number: n, Body: body, Labels: labels}, true
}

// seedPushIssues gives gh the pre-push gate's reads of repo's issues, so
// local-branch reads none of them again. Outside pre-push it adds nothing.
func seedPushIssues(gh aftereffect.Client, repo string) {
	if os.Getenv("FITNESS_INTERNAL_HOOK") != "pre-push" {
		return
	}
	var issues []PushIssue
	if json.Unmarshal([]byte(os.Getenv(PushIssuesEnv)), &issues) != nil {
		return
	}
	for _, issue := range issues {
		if strings.EqualFold(issue.Repo, repo) {
			gh.Seed(issue.Number, issue.Body, issue.Labels)
		}
	}
}
