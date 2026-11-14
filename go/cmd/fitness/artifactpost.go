package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// artifactTool is the Claude Code tool that publishes a page as an
// Artifact.
const artifactTool = "Artifact"

// artifactChecks are the checks a page meets before it publishes.
var artifactChecks = []string{"open-questions"}

// artifactInput is the page an Artifact call publishes and the edits it
// makes to it first. Asset is set when the call uploads a file instead.
type artifactInput struct {
	FilePath string `json:"file_path"`
	Asset    bool   `json:"asset"`
	Edits    []struct {
		Old string `json:"old_str"`
		New string `json:"new_str"`
	} `json:"edits"`
}

// artifactPosts is the page an Artifact call would publish, edits made.
// An asset upload or an unreadable page posts nothing, and the tool refuses
// the latter itself.
func artifactPosts(in claudeHookInput) []ghPost {
	var event struct {
		Input artifactInput `json:"tool_input"`
	}
	_ = json.Unmarshal(in.raw, &event)
	page, ok := event.Input.page(in.Cwd)
	if !ok {
		return nil
	}
	part := postPart{subject: "the artifact page " + filepath.Base(event.Input.FilePath), text: page}
	return []ghPost{{checks: artifactChecks, dir: in.Cwd, parts: []postPart{part}}}
}

// page is the text the call publishes, its edits made. It is false when the
// call publishes no page the hook can read.
func (input artifactInput) page(cwd string) (string, bool) {
	if input.Asset || input.FilePath == "" {
		return "", false
	}
	path := input.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	raw, err := os.ReadFile(path)
	page := string(raw)
	for _, e := range input.Edits {
		page = strings.Replace(page, e.Old, e.New, 1)
	}
	return page, err == nil
}
