package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// runComments runs `fitness comments [path...]`: it prints each comment
// block of the named files, or of every tracked file, one per line as
// `path:line: language kind: text`, the block's lines joined by spaces.
func runComments(args []string) int {
	blocks, err := readComments(args)
	if err != nil {
		report.Error("fitness comments", err)
		return 1
	}
	for _, b := range blocks {
		fmt.Printf("%s:%d: %s %s: %s\n", b.Path, b.Line, b.Lang, b.Kind, strings.Join(strings.Fields(b.Text), " "))
	}
	return 0
}

// readComments reads the blocks of the files args name, relative to the
// working directory, or of every tracked file when args is empty.
func readComments(args []string) ([]comments.Block, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	files := make([]string, len(args))
	for i, arg := range args {
		files[i] = filepath.ToSlash(filepath.Clean(arg))
	}
	if len(files) == 0 {
		if files, err = walkfs.AllFiles(root); err != nil {
			return nil, err
		}
	}
	return comments.Files(root, files)
}
