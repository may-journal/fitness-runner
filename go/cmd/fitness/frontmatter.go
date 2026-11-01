package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/frontmatter"
	"github.com/may-journal/fitness-runner/go/internal/report"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// runStripFrontMatter runs `fitness strip-front-matter`: it deletes fitness keys
// from tracked Markdown front matter, dropping empty blocks, and prints changed files.
func runStripFrontMatter(args []string) int {
	root, err := os.Getwd()
	if err == nil && len(args) > 0 {
		err = fmt.Errorf("takes no arguments, got %s", strings.Join(args, " "))
	}
	if err != nil {
		report.Error("fitness strip-front-matter", err)
		return 1
	}
	stripped, err := stripAll(root)
	if err != nil {
		report.Error("fitness strip-front-matter", err)
		return 1
	}
	fmt.Printf("fitness strip-front-matter: stripped %d files\n", stripped)
	return 0
}

// stripAll strips each Markdown file under root and counts the files it
// changed.
func stripAll(root string) (int, error) {
	stripped := 0
	for _, rel := range walkfs.FilesByExt(root, ".md") {
		changed, err := stripFile(root, rel)
		if err != nil {
			return stripped, err
		}
		if changed {
			stripped++
		}
	}
	return stripped, nil
}

// stripFile strips one file, prints what it did, and reports whether it
// changed the file.
func stripFile(root, rel string) (bool, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	body, others := frontmatter.Strip(string(data))
	if body == string(data) {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return false, err
	}
	line := "stripped " + rel
	if len(others) > 0 {
		line += ": front matter keeps " + strings.Join(others, ", ")
	}
	fmt.Println(line)
	return true, nil
}
