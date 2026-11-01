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

// runComments runs `fitness comments [path...]`, printing each comment block
// of the tracked or named files as `path:line: language kind: text`.
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

// readComments reads the blocks of the tracked files args name, or of
// every tracked file when args is empty.
func readComments(args []string) ([]comments.Block, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	files, err := walkfs.AllFiles(root)
	if err == nil && len(args) > 0 {
		files, err = named(root, files, args)
	}
	if err != nil {
		return nil, err
	}
	return comments.Files(root, files)
}

// named keeps the tracked files each arg names: the file itself, or the
// files under a folder. An arg naming no tracked file is an error.
func named(root string, tracked, args []string) ([]string, error) {
	var out []string
	for _, arg := range args {
		picked := under(tracked, relArg(root, arg))
		if len(picked) == 0 {
			return nil, fmt.Errorf("%s names no tracked file", arg)
		}
		out = append(out, picked...)
	}
	return out, nil
}

// relArg is arg relative to root, slash-separated; an absolute arg is
// resolved through symlinks first, as root is.
func relArg(root, arg string) string {
	if filepath.IsAbs(arg) {
		if rel, err := filepath.Rel(realPath(root), realPath(arg)); err == nil {
			arg = rel
		}
	}
	return filepath.ToSlash(filepath.Clean(arg))
}

// under keeps the tracked files at rel or below it.
func under(tracked []string, rel string) []string {
	var out []string
	for _, f := range tracked {
		if rel == "." || f == rel || strings.HasPrefix(f, rel+"/") {
			out = append(out, f)
		}
	}
	return out
}
