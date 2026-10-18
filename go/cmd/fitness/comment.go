package main

import (
	"errors"
	"os"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/metadata"
	"github.com/may-journal/fitness-runner/go/internal/report"
)

// runCommentCheck judges a drafted comment from --file or stdin, the way an
// agent checks it before posting. With no --file inside GitHub Actions, it
// judges the event's comments on GitHub instead.
func runCommentCheck(args []string) int {
	path, given, err := commentFile(args)
	if err != nil {
		report.Error("fitness comment-check", err)
		return usageExit
	}
	if !given && os.Getenv("GITHUB_EVENT_NAME") != "" {
		return metadata.RunGH("comment-check")
	}
	if err := readFrom(path); err != nil {
		report.Error("fitness comment-check", err)
		return usageExit
	}
	return run([]string{blobFlag, "-", "--checks=" + strings.Join(metadata.CommentChecks, ",")})
}

// readFrom points stdin at path, which the blob runner reads; "-" keeps
// stdin. The file stays open until the process exits.
func readFrom(path string) error {
	if path == "-" {
		return nil
	}
	f, err := os.Open(path)
	if err == nil {
		os.Stdin = f
	}
	return err
}

// commentFile reads `--file <path>` or `--file=<path>`; no flag means stdin.
func commentFile(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "-", false, nil
	}
	if path, ok := fileFlag(args); ok {
		return path, true, nil
	}
	return "", false, errors.New("use --file <path>, --file - for stdin, or no arguments")
}

// fileFlag is the value of a lone `--file <path>` or `--file=<path>`.
func fileFlag(args []string) (string, bool) {
	if len(args) == 2 && args[0] == "--file" {
		return args[1], true
	}
	path, ok := strings.CutPrefix(args[0], "--file=")
	return path, ok && len(args) == 1
}
