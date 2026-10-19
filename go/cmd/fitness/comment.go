package main

import (
	"errors"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// commentChecks selects the checks comment-check runs on a draft.
var commentChecks = "--checks=" + strings.Join(metadata.CommentChecks, ",")

// runCommentCheck judges a drafted comment given as `--blob <text>`, or
// `--blob -` and no arguments for stdin, the way an agent checks it before
// posting. With no arguments inside GitHub Actions, it judges the event's
// comments on GitHub instead.
func runCommentCheck(args []string) int {
	if len(args) > 0 {
		return draftCheck(args)
	}
	if os.Getenv("GITHUB_EVENT_NAME") != "" {
		return runGH("comment-check")
	}
	return run([]string{blobFlag, "-", commentChecks})
}

// draftCheck judges the draft that args name with --blob; any other argument
// is a usage error.
func draftCheck(args []string) int {
	if !hasBlobFlag(args) {
		report.Error("fitness comment-check", errors.New("use --blob <text>, --blob - for stdin, or no arguments"))
		return usageExit
	}
	return run(append(args, commentChecks))
}
