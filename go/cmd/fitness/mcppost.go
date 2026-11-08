package main

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/metadata"
)

// mcpTool is a GitHub MCP tool that posts a body: the checks GitHub gives
// it, and whether that body is an issue's.
type mcpTool struct {
	checks []string
	issue  bool
}

// mcpTools are the GitHub MCP server's posting tools, by the name after the
// server's prefix.
var mcpTools = map[string]mcpTool{
	"add_issue_comment":                     {checks: metadata.CommentChecks},
	"add_comment_to_pending_review":         {checks: metadata.CommentChecks},
	"add_reply_to_pull_request_comment":     {checks: metadata.CommentChecks},
	"create_pull_request_review":            {checks: metadata.CommentChecks},
	"create_and_submit_pull_request_review": {checks: metadata.CommentChecks},
	"submit_pending_pull_request_review":    {checks: metadata.CommentChecks},
	"pull_request_review_write":             {checks: metadata.CommentChecks},
	"create_pull_request":                   {checks: metadata.PRBodyChecks},
	"update_pull_request":                   {checks: metadata.PRBodyChecks},
	"create_issue":                          {issue: true},
	"update_issue":                          {issue: true},
	"issue_write":                           {issue: true},
}

// mcpInput is the part of a GitHub MCP tool's input that holds what it
// posts and where. Each field is read alone, so one of another type in some
// other tool never fails the hook.
type mcpInput struct {
	Body, Owner, Repo, Title string
	Labels                   []string
	Comments                 []string
}

// mcpPosts is the post a GitHub MCP tool would make, or none for another
// server's tool or one with no body.
func mcpPosts(in claudeHookInput) []ghPost {
	server, name, _ := strings.Cut(strings.TrimPrefix(in.Tool, "mcp__"), "__")
	tool, ok := mcpTools[name]
	if !strings.HasPrefix(in.Tool, "mcp__") || !strings.Contains(server, "github") || !ok {
		return nil
	}
	input := readMCPInput(in.raw)
	parts := composeParts(nil, input.Comments)
	parts = append([]postPart{{subject: "the " + strings.ReplaceAll(name, "_", " ") + " body", text: input.Body}}, parts...)
	parts = slices.DeleteFunc(parts, emptyPart)
	if len(parts) == 0 {
		return nil
	}
	return []ghPost{{checks: tool.checksFor(input), dir: in.Cwd, repo: input.repo(), parts: parts}}
}

// repo is the owner/repo the tool posts to, or "" when it names none.
func (input mcpInput) repo() string {
	if input.Owner == "" || input.Repo == "" {
		return ""
	}
	return strings.ToLower(input.Owner + "/" + input.Repo)
}

// checksFor is the checks the tool's body gets: an issue's by its kind label
// or title, else the tool's own.
func (tool mcpTool) checksFor(input mcpInput) []string {
	if !tool.issue {
		return tool.checks
	}
	return kindChecks(cleanKinds(append(slices.Clone(input.Labels), titleKind.FindString(input.Title))))
}

// readMCPInput reads the fields of an MCP tool's input the hook judges,
// leaving out any of another type.
func readMCPInput(raw []byte) mcpInput {
	var event struct {
		Input map[string]json.RawMessage `json:"tool_input"`
	}
	_ = json.Unmarshal(raw, &event)
	var input mcpInput
	for key, field := range map[string]*string{"body": &input.Body, "owner": &input.Owner, "repo": &input.Repo, "title": &input.Title} {
		_ = json.Unmarshal(event.Input[key], field)
	}
	_ = json.Unmarshal(event.Input["labels"], &input.Labels)
	var comments []struct {
		Body string `json:"body"`
	}
	_ = json.Unmarshal(event.Input["comments"], &comments)
	for _, c := range comments {
		input.Comments = append(input.Comments, c.Body)
	}
	return input
}
