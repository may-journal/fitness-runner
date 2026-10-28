// Command adr-0003 keeps ADR 0003: each workflow job that runs steps
// writes a job summary.
package main

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/check"
)

// steps matches a job's steps key.
var steps = regexp.MustCompile(`(?m)^\s+steps:`)

// Rule is the check.
var Rule = check.Check{Run: run}

func main() { check.Main(Rule) }

func run(ctx *check.Context) (check.Result, error) {
	var files []string
	for _, f := range ctx.Files(".yml") {
		if path.Dir(f) == ".github/workflows" {
			files = append(files, f)
		}
	}
	var found []check.Finding
	for _, f := range files {
		text, err := ctx.Read(f)
		if err != nil {
			return check.Result{}, err
		}
		found = append(found, summaries(f, text)...)
	}
	return check.Found(len(files), found...), nil
}

// summaries fails a workflow whose jobs with steps outnumber its lines that
// write the job summary.
func summaries(f, text string) []check.Finding {
	jobs := len(steps.FindAllString(text, -1))
	written := strings.Count(text, "GITHUB_STEP_SUMMARY")
	if written >= jobs {
		return nil
	}
	return []check.Finding{{File: f, Rule: "job-summary",
		Message: fmt.Sprintf("%d jobs run steps, and only %d lines write GITHUB_STEP_SUMMARY (ADR 0003, decision 4)", jobs, written),
		Fix:     "give each job a step that writes its outcome to the job summary"}}
}
